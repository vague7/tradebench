import { useEffect, useRef, useState } from 'react';
import type { SubmissionStatus } from '../types/api';

interface LogEntry {
  time: string;
  message: string;
  level: 'info' | 'success' | 'error';
}

const STATUS_MESSAGES: Partial<Record<SubmissionStatus, { msg: string; level: 'info' | 'success' | 'error' }>> = {
  UPLOADED: { msg: 'ZIP uploaded and validated', level: 'info' },
  BUILDING: { msg: 'Docker image build started', level: 'info' },
  RUNNING: { msg: 'Container started, health check passed', level: 'info' },
  BENCHMARKING: { msg: 'Benchmark load test running', level: 'info' },
  SCORED: { msg: 'Benchmark complete — score computed', level: 'success' },
  FAILED: { msg: 'Pipeline failed', level: 'error' },
};

interface EventLogProps {
  status: SubmissionStatus | null;
  submissionId: string | null;
}

function now(): string {
  return new Date().toLocaleTimeString('en-US', { hour12: false });
}

export function EventLog({ status, submissionId }: EventLogProps) {
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const prevStatusRef = useRef<SubmissionStatus | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  // Add log entry when status changes
  useEffect(() => {
    if (!status || status === prevStatusRef.current) return;
    prevStatusRef.current = status;

    const info = STATUS_MESSAGES[status];
    if (info) {
      setEntries((prev) => [...prev, { time: now(), message: info.msg, level: info.level }]);
    }
  }, [status]);

  // Add initial entry when submission is created
  useEffect(() => {
    if (submissionId) {
      setEntries([{ time: now(), message: `Submission ${submissionId.slice(0, 8)}… created`, level: 'info' }]);
      prevStatusRef.current = null;
    } else {
      setEntries([]);
      prevStatusRef.current = null;
    }
  }, [submissionId]);

  // Auto-scroll to bottom
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [entries]);

  if (entries.length === 0) return null;

  return (
    <div className="event-log panel">
      <h4 className="section-label">Event Log</h4>
      <div className="event-log-scroll">
        {entries.map((e, i) => (
          <div key={i} className={`event-line event-line--${e.level}`}>
            <span className="event-time">{e.time}</span>
            <span className="event-msg">{e.message}</span>
          </div>
        ))}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}
