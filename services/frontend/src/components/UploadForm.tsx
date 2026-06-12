import { type DragEvent, type FormEvent, useRef, useState } from 'react';
import { showToast } from './Toast';

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

const MAX_SIZE_MB = 50;
const MAX_SIZE_BYTES = MAX_SIZE_MB * 1024 * 1024;

export function UploadForm({ onSubmit, loading, error }: UploadFormProps) {
  const [teamName, setTeamName] = useState('');
  const [token, setToken] = useState('');
  const [showToken, setShowToken] = useState(false);
  const [zipFile, setZipFile] = useState<File | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const validateFile = (file: File): string | null => {
    if (!file.name.endsWith('.zip')) return 'Only .zip files are accepted.';
    if (file.size > MAX_SIZE_BYTES) return `File exceeds ${MAX_SIZE_MB} MB limit.`;
    return null;
  };

  const acceptFile = (file: File) => {
    const err = validateFile(file);
    if (err) {
      setValidationError(err);
      showToast('error', err);
      return;
    }
    setZipFile(file);
    setValidationError(null);
  };

  const handleDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer.files[0];
    if (file) acceptFile(file);
  };

  const handleDragOver = (e: DragEvent<HTMLDivElement>) => { e.preventDefault(); setDragOver(true); };
  const handleDragLeave = () => setDragOver(false);
  const handleBrowse = () => fileInputRef.current?.click();

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) acceptFile(file);
  };

  const canSubmit = teamName.trim() !== '' && token.trim() !== '' && zipFile !== null && !loading;

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setValidationError(null);
    if (!teamName.trim()) { setValidationError('Team name is required.'); return; }
    if (!token.trim()) { setValidationError('Bearer token is required.'); return; }
    if (!zipFile) { setValidationError('Drop a ZIP file first.'); return; }
    const fileErr = validateFile(zipFile);
    if (fileErr) { setValidationError(fileErr); return; }
    onSubmit({ teamName: teamName.trim(), token: token.trim(), zipFile });
  };

  const displayError = validationError ?? error;
  const fileSizeMB = zipFile ? (zipFile.size / (1024 * 1024)).toFixed(2) : null;

  return (
    <form className="panel upload-form" onSubmit={handleSubmit} id="upload-form">
      <h3 className="section-label">Upload Submission</h3>

      {/* Requirement chips */}
      <div className="req-chips">
        <span className="req-chip">.zip only</span>
        <span className="req-chip">Max 50 MB</span>
        <span className="req-chip">Dockerfile at root</span>
        <span className="req-chip">Bearer token</span>
      </div>

      {/* Drop zone */}
      <div
        className={`drop-zone${dragOver ? ' drop-zone--over' : ''}${zipFile ? ' drop-zone--filled' : ''}${loading ? ' drop-zone--disabled' : ''}`}
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onClick={loading ? undefined : handleBrowse}
        role="button"
        tabIndex={0}
        aria-label="Drop ZIP file here or click to browse"
        onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') handleBrowse(); }}
      >
        <input
          ref={fileInputRef}
          type="file"
          accept=".zip"
          onChange={handleFileChange}
          hidden
          disabled={loading}
        />
        {zipFile ? (
          <div className="drop-zone-file">
            <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
              <path d="M4 1.5h7l3.5 3.5V16a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V2.5A1 1 0 0 1 4 1.5z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/>
              <path d="M11 1.5v3.5h3.5" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/>
            </svg>
            <div className="drop-zone-meta">
              <span className="drop-zone-name">{zipFile.name}</span>
              <span className="drop-zone-size">{fileSizeMB} MB</span>
            </div>
            {!loading && (
              <button
                type="button"
                className="drop-zone-clear"
                onClick={(e) => { e.stopPropagation(); setZipFile(null); }}
                aria-label="Remove file"
              >
                ×
              </button>
            )}
          </div>
        ) : (
          <div className="drop-zone-empty">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
              <path d="M12 4v12M6 10l6-6 6 6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
              <path d="M4 18v2a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1v-2" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
            </svg>
            <p className="drop-zone-text">
              Drop <strong>.zip</strong> here or <span className="drop-zone-link">browse</span>
            </p>
          </div>
        )}
      </div>

      {/* Upload progress */}
      {loading && (
        <div className="upload-progress">
          <div className="upload-progress-bar" />
        </div>
      )}

      {/* Fields */}
      <label className="form-field">
        <span className="field-label">Team Name</span>
        <input
          id="team-name-input"
          value={teamName}
          onChange={(e) => setTeamName(e.target.value)}
          placeholder="e.g. npcomplete"
          disabled={loading}
          autoComplete="off"
        />
      </label>

      <label className="form-field">
        <span className="field-label">Bearer Token</span>
        <div className="input-with-action">
          <input
            id="team-token-input"
            type={showToken ? 'text' : 'password'}
            value={token}
            onChange={(e) => setToken(e.target.value)}
            placeholder="••••••••"
            disabled={loading}
            autoComplete="off"
          />
          <button
            type="button"
            className="input-toggle"
            onClick={() => setShowToken(!showToken)}
            aria-label={showToken ? 'Hide token' : 'Show token'}
            tabIndex={-1}
          >
            {showToken ? (
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M2 8s2.5-4 6-4 6 4 6 4-2.5 4-6 4-6-4-6-4z" stroke="currentColor" strokeWidth="1.3"/><circle cx="8" cy="8" r="2" stroke="currentColor" strokeWidth="1.3"/></svg>
            ) : (
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M2 8s2.5-4 6-4 6 4 6 4-2.5 4-6 4-6-4-6-4z" stroke="currentColor" strokeWidth="1.3"/><circle cx="8" cy="8" r="2" stroke="currentColor" strokeWidth="1.3"/><path d="M3 13L13 3" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/></svg>
            )}
          </button>
        </div>
      </label>

      <button
        id="upload-submit-btn"
        type="submit"
        disabled={!canSubmit}
        className={canSubmit ? 'submit-btn' : 'submit-btn submit-btn--disabled'}
      >
        {loading ? (
          <span className="btn-loading">
            <span className="spinner" /> Uploading…
          </span>
        ) : (
          'Upload submission'
        )}
      </button>

      {/* Validation messages */}
      {displayError && (
        <div className="form-error-inline" role="alert">
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
            <circle cx="7" cy="7" r="6" stroke="currentColor" strokeWidth="1.3"/>
            <path d="M7 4.5V7.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
            <circle cx="7" cy="9.5" r="0.7" fill="currentColor"/>
          </svg>
          {displayError}
        </div>
      )}

      {!canSubmit && !loading && !displayError && (
        <p className="form-hint">
          {!teamName.trim() ? 'Enter team name' : !token.trim() ? 'Enter bearer token' : !zipFile ? 'Select a ZIP file' : ''}
        </p>
      )}
    </form>
  );
}
