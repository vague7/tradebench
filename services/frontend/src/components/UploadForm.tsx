import { type FormEvent, useState } from 'react';

export interface UploadFormData {
  teamName: string;
  token: string;
  zipFile: File;
}

interface UploadFormProps {
  onSubmit: (data: UploadFormData) => void;
  loading: boolean;
  error: string | null;
}

export function UploadForm({ onSubmit, loading, error }: UploadFormProps) {
  const [teamName, setTeamName] = useState('');
  const [token, setToken] = useState('');
  const [zipFile, setZipFile] = useState<File | null>(null);
  const [validationError, setValidationError] = useState<string | null>(null);

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setValidationError(null);

    if (!teamName.trim()) {
      setValidationError('Team name is required.');
      return;
    }
    if (!token.trim()) {
      setValidationError('Team token is required.');
      return;
    }
    if (!zipFile) {
      setValidationError('Select a ZIP file first.');
      return;
    }
    if (!zipFile.name.endsWith('.zip')) {
      setValidationError('Only .zip files are accepted.');
      return;
    }

    onSubmit({ teamName: teamName.trim(), token: token.trim(), zipFile });
  };

  const displayError = validationError ?? error;

  return (
    <form className="panel upload-form" onSubmit={handleSubmit} id="upload-form">
      <h3 className="form-title">Submit your exchange</h3>
      <label>
        <span className="field-label">Team name</span>
        <input
          id="team-name-input"
          value={teamName}
          onChange={(e) => setTeamName(e.target.value)}
          placeholder="Team Alpha"
          disabled={loading}
        />
      </label>
      <label>
        <span className="field-label">Team token</span>
        <input
          id="team-token-input"
          type="password"
          value={token}
          onChange={(e) => setToken(e.target.value)}
          placeholder="Bearer token"
          disabled={loading}
        />
      </label>
      <label>
        <span className="field-label">Submission ZIP</span>
        <input
          id="zip-file-input"
          type="file"
          accept=".zip"
          onChange={(e) => setZipFile(e.target.files?.[0] ?? null)}
          disabled={loading}
        />
      </label>
      <button id="upload-submit-btn" type="submit" disabled={loading}>
        {loading ? 'Uploading…' : 'Upload submission'}
      </button>
      {displayError ? <p className="form-error" role="alert">{displayError}</p> : null}
    </form>
  );
}
