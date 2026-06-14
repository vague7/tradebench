# Changelog

## 1. Load Testing Target Increased
*   **File Changed**: `.env`
*   **Detail**: Modified `BOT_DEFAULT_COUNT` from `500` to `10000` to significantly increase the concurrent load tested against the sandbox engine and telemetry systems.

## 2. Frontend Leaderboard Logic Fixed
*   **File Changed**: `services/frontend/src/components/LeaderboardTable.tsx`
*   **Detail**: 
    *   Updated the filtering logic so that submissions failing at the Docker build or sandbox launch stage (`status === 'FAILED'`) are completely removed from the "Main Rankings" tab.
    *   Ensured these failed submissions correctly populate only in the "Failing" tab.
    *   Updated the "Reason" column in the Failing tab to correctly display "Build/Runtime Failed" instead of incorrectly defaulting to "Disqualified" for broken uploads.

## 3. Frontend Dockerfile Cache Optimization
*   **File Changed**: `services/frontend/Dockerfile`
*   **Detail**: Moved the `RUN npm install` layer above the `COPY src ./src` and `COPY public ./public` layers. This prevents Docker from busting the entire `npm install` cache whenever a React source file is modified, reducing rebuild times from 5+ minutes to mere seconds.

## 4. Test Submissions Added
*   **Detail**: Created two mock submissions for testing the platform:
    *   **Passing Mock**: Source files in `test2_submission/` folder, zipped as `test2.zip` (located in the root `tradebench` directory). This is a fast, properly formatted server for testing the 10,000 bot load.
    *   **Failing Mock**: Source files in `failed_submission/` folder, zipped as `failed.zip` (located in the root `tradebench` directory). This contains an intentional unused import error to test the Failing tab logic on the leaderboard.
