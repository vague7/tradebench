import { useCallback, useEffect, useState } from 'react';

import { getSubmissionResults, uploadSubmission } from '../api/client';
import { MetricsPanel } from '../components/MetricsPanel';
import { StatusBadge } from '../components/StatusBadge';
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
    } catch (err) {
      setUploadError(err instanceof Error ? err.message : 'Upload failed');
    } finally {
      setUploadLoading(false);
    }
  }, []);

  // Auto-fetch results when submission reaches SCORED.
  useEffect(() => {
    if (phase !== 'success' || !submissionId || !teamToken) return;

    let cancelled = false;
    const fetchResults = async () => {
      try {
        const results = await getSubmissionResults(submissionId, teamToken);
        if (!cancelled) {
          setSnapshot(results.snapshot);
          setScore(results.score);
        }
      } catch {
        // Silently handle — results may not be available immediately.
      }
    };
    void fetchResults();
    return () => { cancelled = true; };
  }, [phase, submissionId, teamToken]);

  const phaseLabel = (): string => {
    switch (phase) {
      case 'idle': return 'Awaiting submission';
      case 'loading': return 'Connecting to backend…';
      case 'polling': return 'Polling status updates…';
      case 'success': return 'Benchmark complete';
      case 'failed': return 'Benchmark failed';
      case 'timeout': return 'Polling timed out';
    }
  };

  return (
    <div className="page-grid">
      <section className="hero-panel">
        <div className="aurora-bg-container">
          <div className="aurora-grid"></div>
        </div>
        <div className="hero-content">
          <p className="eyebrow">Submission pipeline</p>
          <h1>Upload your trading exchange and watch it get benchmarked in real time.</h1>
          <p className="lead">
            Submit a ZIP containing your Dockerfile and source code. The platform
            will build, deploy, stress-test, and score your implementation automatically.
          </p>

          <div className="hero-status">
            <span className="status-label">Pipeline phase</span>
            <p className="phase-label">{phaseLabel()}</p>
            {submission ? <StatusBadge status={submission.status} /> : null}
          </div>

          {submission ? (
            <dl className="status-grid">
              <div>
                <dt>Submission</dt>
                <dd>{submission.id}</dd>
              </div>
              <div>
                <dt>Team</dt>
                <dd>{submission.teamName}</dd>
              </div>
              <div>
                <dt>Uploaded</dt>
                <dd>{new Date(submission.uploadedAt).toLocaleString()}</dd>
              </div>
            </dl>
          ) : null}

          {pollError ? <p className="form-error" role="alert">{pollError}</p> : null}
        </div>
      </section>

      <div className="stack">
        <UploadForm onSubmit={handleUpload} loading={uploadLoading} error={uploadError} />
        <MetricsPanel snapshot={snapshot} score={score} />
      </div>
    </div>
  );
}
