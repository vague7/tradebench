import { useCallback, useEffect, useRef, useState } from 'react';

import { getSubmissionHistory, getSubmissionResults, uploadSubmission } from '../api/client';

import { CopyButton } from '../components/CopyButton';
import { ErrorBanner } from '../components/ErrorBanner';
import { EventLog } from '../components/EventLog';
import { DiagnosisPanel } from '../components/DiagnosisPanel';
import { MetricsChart } from '../components/MetricsChart';
import { MetricsPanel } from '../components/MetricsPanel';
import { SubmissionPipeline } from '../components/PipelineTracker';
import { StatusPill } from '../components/StatusPill';
import { showToast } from '../components/Toast';
import { UploadForm } from '../components/UploadForm';
import type { UploadFormData } from '../components/UploadForm';
import { useSubmissionStatus } from '../hooks/useSubmissionStatus';
import type { MetricSnapshot, Score } from '../types/api';


export function SubmitPage() {
  const [submissionId, setSubmissionId] = useState<string | null>(() => localStorage.getItem('bench_submission_id'));
  const [teamToken, setTeamToken] = useState(() => localStorage.getItem('bench_team_token') || '');
  const [uploadLoading, setUploadLoading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);

  const { submission, phase, error: pollError } = useSubmissionStatus(submissionId, teamToken);

  const [snapshot, setSnapshot] = useState<MetricSnapshot | null>(null);
  const [score, setScore] = useState<Score | null>(null);
  const [history, setHistory] = useState<MetricSnapshot[]>([]);

  // Latch: once we have a submission object, keep it visible even during
  // brief phase transitions so the UI never goes blank.
  const lastSubmissionRef = useRef<typeof submission>(null);
  if (submission !== null) lastSubmissionRef.current = submission;
  const displaySubmission = submission ?? lastSubmissionRef.current;

  const handleUpload = useCallback(async (data: UploadFormData) => {
    setUploadLoading(true);
    setUploadError(null);
    try {
      const result = await uploadSubmission(data);
      setSubmissionId(result.submissionId);
      setTeamToken(data.token);
      localStorage.setItem('bench_submission_id', result.submissionId);
      localStorage.setItem('bench_team_token', data.token);
      setSnapshot(null);
      setScore(null);
      showToast('success', `Submission created: ${result.submissionId.slice(0, 8)}…`);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Upload failed';
      setUploadError(msg);
      showToast('error', msg);
    } finally {
      setUploadLoading(false);
    }
  }, []);

  const handleReset = useCallback(() => {
    setSubmissionId(null);
    setTeamToken('');
    localStorage.removeItem('bench_submission_id');
    localStorage.removeItem('bench_team_token');
    localStorage.removeItem('bench_rca_history');
    setUploadError(null);
    setSnapshot(null);
    setScore(null);
    setHistory([]);
  }, []);

  // Metrics polling: driven by submissionId + teamToken only.
  // phase is read via ref so changing phase never tears this effect down mid-flight.
  const phaseRef = useRef(phase);
  phaseRef.current = phase;

  useEffect(() => {
    if (!submissionId || !teamToken) return;

    let cancelled = false;

    const tick = async () => {
      if (cancelled) return;
      const currentPhase = phaseRef.current;
      if (currentPhase !== 'polling' && currentPhase !== 'success') return;

      const [resultsResult, historyResult] = await Promise.allSettled([
        getSubmissionResults(submissionId, teamToken),
        getSubmissionHistory(submissionId, teamToken),
      ]);

      if (cancelled) return;

      if (resultsResult.status === 'fulfilled' && resultsResult.value.snapshot) {
        setSnapshot(resultsResult.value.snapshot);
        if (resultsResult.value.score) setScore(resultsResult.value.score);
      }
      if (historyResult.status === 'fulfilled' && historyResult.value?.length > 0) {
        setHistory(historyResult.value);
      }
    };

    void tick();
    const timer = setInterval(() => void tick(), 2000);
    return () => { cancelled = true; clearInterval(timer); };
  }, [submissionId, teamToken]); // phase intentionally excluded — read via ref

  // Auto-reset only for the truly stale-ID case: no submission data at all
  // and we've been idle/failed for a sustained period.
  useEffect(() => {
    if (phase !== 'failed' && phase !== 'timeout') return;
    if (displaySubmission !== null) return; // still have data, don't wipe
    if (!submissionId) return;
    const t = setTimeout(() => handleReset(), 5000);
    return () => clearTimeout(t);
  }, [phase, displaySubmission, submissionId, handleReset]);

  const isTerminal = phase === 'success' || phase === 'failed' || phase === 'timeout';
  const hasSubmission = displaySubmission !== null;

  return (
    <div className="submit-page">
      <div className="submit-layout">
        {/* Left column */}
        <div className="submit-left">
          {!hasSubmission && (
            <UploadForm onSubmit={handleUpload} loading={uploadLoading} error={uploadError} />
          )}

          {hasSubmission && (
            <div className="panel sub-card" id="submission-details">
              <h3 className="section-label">Submission Details</h3>
              <div className="sub-row">
                <span className="sub-key">ID</span>
                <span className="sub-val mono">
                  {displaySubmission!.id.slice(0, 12)}…
                  <CopyButton text={displaySubmission!.id} label="Copy submission ID" />
                </span>
              </div>
              <div className="sub-row">
                <span className="sub-key">Team</span>
                <span className="sub-val">{displaySubmission!.teamName}</span>
              </div>
              <div className="sub-row">
                <span className="sub-key">Status</span>
                <StatusPill status={displaySubmission!.status} />
              </div>
              <div className="sub-row">
                <span className="sub-key">Uploaded</span>
                <span className="sub-val mono">{new Date(displaySubmission!.uploadedAt).toLocaleTimeString()}</span>
              </div>
              {displaySubmission!.benchmarkStart && (
                <div className="sub-row">
                  <span className="sub-key">Bench start</span>
                  <span className="sub-val mono">{new Date(displaySubmission!.benchmarkStart).toLocaleTimeString()}</span>
                </div>
              )}
              {displaySubmission!.benchmarkEnd && (
                <div className="sub-row">
                  <span className="sub-key">Bench end</span>
                  <span className="sub-val mono">{new Date(displaySubmission!.benchmarkEnd).toLocaleTimeString()}</span>
                </div>
              )}
              {displaySubmission!.errorMessage && (
                <div className="sub-error">{displaySubmission!.errorMessage}</div>
              )}
              {isTerminal && (
                <button className="reset-btn" onClick={handleReset} type="button" id="reset-btn">
                  ← Submit another
                </button>
              )}
            </div>
          )}

          {/* Event log — only when submission is active */}
          {hasSubmission && (
            <EventLog
              status={displaySubmission!.status}
              submissionId={displaySubmission!.id}
              history={history}
              score={score}
            />
          )}
        </div>

        {/* Right column */}
        <div className="submit-right">
          <SubmissionPipeline currentStatus={hasSubmission ? displaySubmission!.status : null} />

          {(phase === 'failed' || phase === 'timeout') && (
            <ErrorBanner
              title={phase === 'failed' ? 'Benchmark Failed' : 'Polling Timed Out'}
              message={displaySubmission?.errorMessage ?? pollError ?? 'An unexpected error occurred during the benchmark pipeline.'}
              detail={pollError ?? undefined}
              onRetry={handleReset}
            />
          )}

          <MetricsPanel snapshot={snapshot} score={score} />

          {history.length >= 2 && (
            <MetricsChart snapshots={history} />
          )}
        </div>
      </div>

      <DiagnosisPanel snapshots={history} submissionId={submissionId} />
    </div>
  );
}
