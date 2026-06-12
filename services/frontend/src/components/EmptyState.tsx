interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  description: string;
  action?: {
    label: string;
    onClick: () => void;
  };
}

export function EmptyState({ icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="empty-state">
      {icon && <div className="empty-state-icon">{icon}</div>}
      <p className="empty-state-title">{title}</p>
      <p className="empty-state-desc">{description}</p>
      {action && (
        <button className="empty-state-action" onClick={action.onClick} type="button">
          {action.label}
        </button>
      )}
    </div>
  );
}
