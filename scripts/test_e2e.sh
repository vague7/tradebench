#!/bin/bash
set -e

# Load .env variables if they exist
if [ -f ".env" ]; then
  # shellcheck disable=SC2046
  export $(grep -v '^#' .env | xargs)
fi

API_BASE=${API_BASE:-"http://localhost:8080"}
ADMIN_TOKEN=${ADMIN_TOKEN:-"changeme-admin"}
TEAM_TOKEN="test-team-token"
TEAM_NAME="E2E-Test-Team-$(date +%s)"
TIMEOUT=360
ZIP_FILE="test_sub.zip"

echo "Running E2E Smoke Test against $API_BASE"

# 1. Zip the dummy submission
cd scripts/test_submission
zip -r "../$ZIP_FILE" Dockerfile main.go go.mod > /dev/null
cd ../..

echo "Uploading test submission..."
UPLOAD_RES=$(curl -s -X POST "$API_BASE/api/submissions" \
  -H "Authorization: Bearer $TEAM_TOKEN" \
  -F "teamName=$TEAM_NAME" \
  -F "file=@scripts/$ZIP_FILE")
echo "$UPLOAD_RES"

# Extract submissionId — try python3 first, then python, then grep
SUB_ID=$(python3 -c "import sys, json; print(json.load(sys.stdin).get('submissionId', ''))" <<< "$UPLOAD_RES" 2>/dev/null || true)
if [ -z "$SUB_ID" ]; then
  SUB_ID=$(python -c "import sys, json; print(json.load(sys.stdin).get('submissionId', ''))" <<< "$UPLOAD_RES" 2>/dev/null || true)
fi
if [ -z "$SUB_ID" ]; then
  SUB_ID=$(echo "$UPLOAD_RES" | grep -o '"submissionId":"[^"]*' | cut -d'"' -f4 || true)
fi

if [ -z "$SUB_ID" ]; then
  echo "FAIL: Could not extract submissionId from response: $UPLOAD_RES"
  rm -f "scripts/$ZIP_FILE"
  exit 1
fi
echo "Upload successful. Submission ID: $SUB_ID"

echo "Polling status..."
START_TIME=$(date +%s)
while true; do
  STATUS_RES=$(curl -s -H "Authorization: Bearer $TEAM_TOKEN" "$API_BASE/api/submissions/$SUB_ID/status")
  
  STATUS=$(python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))" <<< "$STATUS_RES" 2>/dev/null || true)
  if [ -z "$STATUS" ]; then
    STATUS=$(python -c "import sys, json; print(json.load(sys.stdin).get('status', ''))" <<< "$STATUS_RES" 2>/dev/null || true)
  fi
  if [ -z "$STATUS" ]; then
    STATUS=$(echo "$STATUS_RES" | grep -o '"status":"[^"]*' | cut -d'"' -f4 || true)
  fi

  if [ "$STATUS" == "SCORED" ]; then
    echo "Status reached SCORED!"
    break
  fi

  if [ "$STATUS" == "FAILED" ]; then
    echo "FAIL: Submission failed during benchmark."
    echo "Response: $STATUS_RES"
    rm -f "scripts/$ZIP_FILE"
    exit 1
  fi

  CURRENT_TIME=$(date +%s)
  ELAPSED=$((CURRENT_TIME - START_TIME))
  if [ "$ELAPSED" -ge "$TIMEOUT" ]; then
    echo "FAIL: Polling timed out after $TIMEOUT seconds."
    rm -f "scripts/$ZIP_FILE"
    exit 1
  fi

  echo "Current status: $STATUS... waiting."
  sleep 5
done

echo "Fetching results..."
RESULTS_RES=$(curl -s -H "Authorization: Bearer $TEAM_TOKEN" "$API_BASE/api/submissions/$SUB_ID/results")
echo "Results response: $RESULTS_RES"

# Extract finalScore — nested under score.finalScore
FINAL_SCORE=$(python3 -c "import sys, json; d=json.load(sys.stdin); print(d.get('score', d).get('finalScore', 0))" <<< "$RESULTS_RES" 2>/dev/null || true)
if [ -z "$FINAL_SCORE" ]; then
  FINAL_SCORE=$(python -c "import sys, json; d=json.load(sys.stdin); print(d.get('score', d).get('finalScore', 0))" <<< "$RESULTS_RES" 2>/dev/null || true)
fi
if [ -z "$FINAL_SCORE" ]; then
  FINAL_SCORE=$(echo "$RESULTS_RES" | grep -o '"finalScore":[0-9.e+-]*' | cut -d':' -f2 || true)
fi

echo "Final score: $FINAL_SCORE"

# Compare as float: pass if > 0 (not empty, not "0", not "0.0", not "0.00...")
SCORE_OK=$(python3 -c "import sys; s='$FINAL_SCORE'; print('yes' if s and float(s) > 0 else 'no')" 2>/dev/null || \
           python  -c "import sys; s='$FINAL_SCORE'; print('yes' if s and float(s) > 0 else 'no')" 2>/dev/null || \
           echo "no")

if [ "$SCORE_OK" != "yes" ]; then
  echo "FAIL: Expected finalScore > 0, got: $FINAL_SCORE"
  echo "Results response: $RESULTS_RES"
  rm -f "scripts/$ZIP_FILE"
  exit 1
fi
echo "Results verified. Final score: $FINAL_SCORE"

echo "Checking leaderboard..."
LEADERBOARD_RES=$(curl -s "$API_BASE/api/leaderboard")
if ! echo "$LEADERBOARD_RES" | grep -q "$TEAM_NAME"; then
  echo "FAIL: Team $TEAM_NAME not found in leaderboard."
  echo "Leaderboard: $LEADERBOARD_RES"
  rm -f "scripts/$ZIP_FILE"
  exit 1
fi

rm -f "scripts/$ZIP_FILE"
echo "SUCCESS: E2E Pipeline works."
exit 0
