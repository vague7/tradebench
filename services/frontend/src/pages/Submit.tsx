import { useMemo, useState } from 'react';

import { getSubmissionResults } from '../api/client';
import { MetricsPanel } from '../components/MetricsPanel';
import { StatusBadge } from '../components/StatusBadge';
import { UploadForm } from '../components/UploadForm';
import { useSubmissionStatus } from '../hooks/useSubmissionStatus';
import type { MetricSnapshot, Score } from '../types/api';

export function SubmitPage() {
  const [submissionId, setSubmissionId] = useState<string | null>(null);
  const [teamToken, setTeamToken] = useState('');
  const { submission, loading, error } = useSubmissionStatus(submissionId, teamToken);
  const [snapshot, setSnapshot] = useState<MetricSnapshot | null>(null);
  const [score, setScore] = useState<Score | null>(null);

  const statusText = useMemo(() => submission?.status ?? 'UPLOADED', [submission?.status]);

  const loadResults = async () => {
    if (!submissionId || !teamToken) {
      return;
    }
    try {
      const result = await getSubmissionResults(submissionId, teamToken);
      setSnapshot(result.snapshot);
      setScore(result.score);
    } catch {
      // Keep the panel resilient while the backend scaffold is still minimal.
    }
  };

  return (
    <div className="page-grid">
      <section className="hero-panel panel">
        <p className="eyebrow">Submission pipeline</p>
        <h1>Upload a contestant ZIP and track it through the benchmark pipeline.</h1>
        <p className="lead">
          This scaffold gives teams a single place to upload, poll status, and inspect the latest score without wiring in any extra routing library.
        </p>
        <div className="hero-status">
          <span className="status-label">Current status</span>
          <StatusBadge status={statusText} />
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
              <dd>{submission.uploadedAt}</dd>
            </div>
          </dl>
        ) : null}
        <button className="secondary-button" type="button" onClick={loadResults} disabled={!submissionId || !teamToken}>
          Refresh results
        </button>
        {loading ? <p className="inline-note">Polling backend status...</p> : null}
        {error ? <p className="form-error">{error}</p> : null}
      </section>

      <div className="stack">
        <UploadForm
          onSubmitted={(nextSubmissionId, token) => {
            setSubmissionId(nextSubmissionId);
            setTeamToken(token);
          }}
        />
        <MetricsPanel snapshot={snapshot} score={score} />
      </div>
    </div>
  );
}
