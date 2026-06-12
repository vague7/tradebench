import { useState } from 'react';

interface ErrorBannerProps {
  title: string;
  message: string;
  detail?: string;
  lastSuccess?: Date | null;
  onRetry?: () => void;
  onDismiss?: () => void;
}

export function ErrorBanner({
  title,
  message,
  detail,
  lastSuccess,
  onRetry,
  onDismiss,
}: ErrorBannerProps) {
  const [showDetail, setShowDetail] = useState(false);

  const lastSuccessText = lastSuccess
    ? lastSuccess.toLocaleTimeString()
    : 'never';

  return (
    <div className="error-banner" role="alert">
      <div className="error-banner-icon">
        <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
          <circle cx="9" cy="9" r="8" stroke="currentColor" strokeWidth="1.5"/>
          <path d="M9 5.5V9.5" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
          <circle cx="9" cy="12.5" r="1" fill="currentColor"/>
        </svg>
      </div>
      <div className="error-banner-body">
        <strong className="error-banner-title">{title}</strong>
        <p className="error-banner-msg">{message}</p>
        <span className="error-banner-meta">Last success: {lastSuccessText}</span>
        {detail && (
          <>
            <button
              className="error-banner-toggle"
              onClick={() => setShowDetail(!showDetail)}
              type="button"
            >
              {showDetail ? 'Hide details' : 'Show details'}
            </button>
            {showDetail && <pre className="error-banner-detail">{detail}</pre>}
          </>
        )}
      </div>
      <div className="error-banner-actions">
        {onRetry && (
          <button className="error-banner-retry" onClick={onRetry} type="button">
            Retry
          </button>
        )}
        {onDismiss && (
          <button className="error-banner-dismiss" onClick={onDismiss} type="button" aria-label="Dismiss">
            ×
          </button>
        )}
      </div>
    </div>
  );
}
