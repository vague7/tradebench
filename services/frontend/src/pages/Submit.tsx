import { useCallback, useEffect, useState } from 'react';

import { getSubmissionResults, uploadSubmission } from '../api/client';
import { BenchmarkPhaseTracker } from '../components/BenchmarkPhaseTracker';
import { CopyButton } from '../components/CopyButton';
import { ErrorBanner } from '../components/ErrorBanner';
import { EventLog } from '../components/EventLog';
import { MetricsPanel } from '../components/MetricsPanel';
import { SubmissionPipeline } from '../components/PipelineTracker';
import { StatusPill } from '../components/StatusPill';
import { showToast } from '../components/Toast';
import { UploadForm } from '../components/UploadForm';
import type { UploadFormData } from '../components/UploadForm';
import { useSubmissionStatus } from '../hooks/useSubmissionStatus';
import type { MetricSnapshot, Score } from '../types/api';

export function SubmitPage() {
  const [submissionId, setSubmissionId] = useState<string | null>(null);
  const [teamToken, setTeamToken] = useState('');
  const [uploadLoading, setUploadLoading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);

  const { submission, phase, error: pollError } = useSubmissionStatus(submissionId, teamToken);

  const [snapshot, setSnapshot] = useState<MetricSnapshot | null>(null);
  const [score, setScore] = useState<Score | null>(null);

  const handleUpload = useCallback(async (data: UploadFormData) => {
    setUploadLoading(true);
    setUploadError(null);
    try {
      const result = await uploadSubmission(data);
      setSubmissionId(result.submissionId);
      setTeamToken(data.token);
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
    setUploadError(null);
    setSnapshot(null);
    setScore(null);
  }, []);

  // Auto-fetch results when scored
  useEffect(() => {
    if (phase !== 'success' || !submissionId || !teamToken) return;
    let cancelled = false;
    const fetchResults = async () => {
      try {
        const results = await getSubmissionResults(submissionId, teamToken);
        if (!cancelled) {
          setSnapshot(results.snapshot);
          setScore(results.score);
          showToast('success', 'Benchmark complete — results ready');
        }
      } catch { /* results may not be immediately available */ }
    };
    void fetchResults();
    return () => { cancelled = true; };
  }, [phase, submissionId, teamToken]);

  const isTerminal = phase === 'success' || phase === 'failed' || phase === 'timeout';
  const hasSubmission = submission !== null;

  return (
    <div className="submit-layout">
      {/* Left column: upload */}
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
                {submission.id.slice(0, 12)}…
                <CopyButton text={submission.id} label="Copy submission ID" />
              </span>
            </div>
            <div className="sub-row">
              <span className="sub-key">Team</span>
              <span className="sub-val">{submission.teamName}</span>
            </div>
            <div className="sub-row">
              <span className="sub-key">Status</span>
              <StatusPill status={submission.status} />
            </div>
            <div className="sub-row">
              <span className="sub-key">Uploaded</span>
              <span className="sub-val mono">{new Date(submission.uploadedAt).toLocaleTimeString()}</span>
            </div>
            {submission.benchmarkStart && (
              <div className="sub-row">
                <span className="sub-key">Bench start</span>
                <span className="sub-val mono">{new Date(submission.benchmarkStart).toLocaleTimeString()}</span>
              </div>
            )}
            {submission.benchmarkEnd && (
              <div className="sub-row">
                <span className="sub-key">Bench end</span>
                <span className="sub-val mono">{new Date(submission.benchmarkEnd).toLocaleTimeString()}</span>
              </div>
            )}
            {submission.errorMessage && (
              <div className="sub-error">{submission.errorMessage}</div>
            )}

            {isTerminal && (
              <button className="reset-btn" onClick={handleReset} type="button" id="reset-btn">
                ← Submit another
              </button>
            )}
          </div>
        )}
      </div>

      {/* Right column: pipeline + metrics */}
      <div className="submit-right">
        {/* Pipeline — always visible */}
        <SubmissionPipeline currentStatus={hasSubmission ? submission.status : null} />

        {/* Error banner */}
        {(phase === 'failed' || phase === 'timeout') && (
          <ErrorBanner
            title={phase === 'failed' ? 'Benchmark Failed' : 'Polling Timed Out'}
            message={submission?.errorMessage ?? pollError ?? 'An unexpected error occurred during the benchmark pipeline.'}
            detail={pollError ?? undefined}
            onRetry={handleReset}
          />
        )}

        {/* Event log — only when submission is active */}
        {hasSubmission && (
          <EventLog status={submission.status} submissionId={submission.id} />
        )}

        {/* Phase tracker preview */}
        <BenchmarkPhaseTracker activePhase={null} />

        {/* Metrics — always visible with placeholders */}
        <MetricsPanel snapshot={snapshot} score={score} />
      </div>
    </div>
  );
}
