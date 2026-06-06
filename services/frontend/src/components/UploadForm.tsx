import { FormEvent, useState } from 'react';

import { uploadSubmission } from '../api/client';

export interface UploadFormState {
  submissionId: string | null;
  error: string | null;
  loading: boolean;
}

export function UploadForm({ onSubmitted }: { onSubmitted: (submissionId: string, token: string) => void }) {
  const [teamName, setTeamName] = useState('');
  const [token, setToken] = useState('');
  const [zipFile, setZipFile] = useState<File | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!zipFile) {
      setError('Select a ZIP file first.');
      return;
    }

    setLoading(true);
    setError(null);
    try {
      const result = await uploadSubmission({ teamName, token, zipFile });
      onSubmitted(result.submissionId, token);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <form className="panel upload-form" onSubmit={handleSubmit}>
      <label>
        Team name
        <input value={teamName} onChange={(event) => setTeamName(event.target.value)} placeholder="Team Alpha" />
      </label>
      <label>
        Team token
        <input value={token} onChange={(event) => setToken(event.target.value)} placeholder="Bearer token" />
      </label>
      <label>
        Submission ZIP
        <input type="file" accept=".zip" onChange={(event) => setZipFile(event.target.files?.[0] ?? null)} />
      </label>
      <button type="submit" disabled={loading}>
        {loading ? 'Uploading…' : 'Upload submission'}
      </button>
      {error ? <p className="form-error">{error}</p> : null}
    </form>
  );
}
