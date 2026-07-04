import { useEffect, useRef, useState } from 'react';
import type { MetricSnapshot, Score, SubmissionStatus } from '../types/api';

interface LogEntry {
  time: string;
  message: string;
  level: 'info' | 'success' | 'error' | 'metric';
}

const STATUS_MESSAGES: Partial<Record<SubmissionStatus, { msg: string; level: 'info' | 'success' | 'error' }>> = {
  UPLOADED:     { msg: 'ZIP received — queued for build', level: 'info' },
  BUILDING:     { msg: 'Docker image build started', level: 'info' },
  RUNNING:      { msg: 'Container launched — awaiting health check', level: 'info' },
  BENCHMARKING: { msg: 'Bot fleet connected — load test active', level: 'info' },
  SCORED:       { msg: 'Benchmark complete — final score computed', level: 'success' },
  FAILED:       { msg: 'Pipeline failed — see error above', level: 'error' },
};

interface EventLogProps {
  status: SubmissionStatus | null;
  submissionId: string | null;
  history?: MetricSnapshot[];
  score?: Score | null;
}

function now(): string {
  return new Date().toLocaleTimeString('en-US', { hour12: false });
}

function fmt(n: number, d = 1): string {
  return n.toFixed(d);
}

export function EventLog({ status, submissionId, history = [], score }: EventLogProps) {
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const prevStatusRef = useRef<SubmissionStatus | null>(null);
  const prevHistoryLenRef = useRef<number>(0);
  const prevScoreRef = useRef<string | null>(null);
  const scrollRef = useRef<HTMLDivElement>(null);

  // Load from localStorage on mount / submissionId change
  useEffect(() => {
    if (submissionId) {
      const saved = localStorage.getItem(`bench_logs_${submissionId}`);
      if (saved) {
        try {
          const parsed: LogEntry[] = JSON.parse(saved);
          setEntries(parsed);
          prevStatusRef.current = status;
          prevHistoryLenRef.current = history.length;
          return;
        } catch { /* ignore */ }
      }
      setEntries([{ time: now(), message: `Submission ${submissionId.slice(0, 8)}… created`, level: 'info' }]);
      prevStatusRef.current = null;
    } else {
      setEntries([]);
      prevStatusRef.current = null;
      prevHistoryLenRef.current = 0;
      prevScoreRef.current = null;
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [submissionId]);

  // Log status transitions
  useEffect(() => {
    if (!status || status === prevStatusRef.current) return;
    prevStatusRef.current = status;
    const info = STATUS_MESSAGES[status];
    if (info) {
      setEntries((prev) => [...prev, { time: now(), message: info.msg, level: info.level }]);
    }
  }, [status]);

  // Log each new metric window that arrives
  useEffect(() => {
    if (history.length === 0) return;
    const newCount = history.length - prevHistoryLenRef.current;
    if (newCount <= 0) return;

    const newWindows = history.slice(prevHistoryLenRef.current);
    prevHistoryLenRef.current = history.length;

    const newEntries: LogEntry[] = newWindows.map((snap) => {
      const total = snap.successCount + snap.failureCount + snap.timeoutCount;
      const errPct = total > 0 ? ((snap.failureCount + snap.timeoutCount) / total) * 100 : 0;
      const t = new Date(snap.windowEnd).toLocaleTimeString('en-US', { hour12: false });
      return {
        time: t,
        message: `W${history.indexOf(snap) + 1}: TPS=${fmt(snap.tps, 0)}  P99=${fmt(snap.p99LatencyMs, 0)}ms  Err=${fmt(errPct, 1)}%  OK=${snap.successCount}  Timeout=${snap.timeoutCount}`,
        level: 'metric',
      };
    });

    setEntries((prev) => [...prev, ...newEntries]);
  }, [history]);

  // Log score updates
  useEffect(() => {
    if (!score) return;
    const key = `${score.finalScore.toFixed(4)}-${score.computedAt}`;
    if (key === prevScoreRef.current) return;
    prevScoreRef.current = key;

    const final = (score.finalScore * 100).toFixed(2);
    const tpt = (score.throughputScore * 100).toFixed(1);
    const lat = (score.latencyScore * 100).toFixed(1);
    const cor = (score.correctnessScore * 100).toFixed(1);
    const dq = score.isDisqualified ? '  ⚠ DISQUALIFIED' : '';
    setEntries((prev) => [
      ...prev,
      {
        time: now(),
        message: `Score updated → ${final}pts  (TPS=${tpt} LAT=${lat} COR=${cor})${dq}`,
        level: score.isDisqualified ? 'error' : 'success',
      },
    ]);
  }, [score]);

  // Persist to localStorage
  useEffect(() => {
    if (submissionId && entries.length > 0) {
      localStorage.setItem(`bench_logs_${submissionId}`, JSON.stringify(entries));
    }
  }, [submissionId, entries]);

  // Auto-scroll
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [entries]);

  if (entries.length === 0) return null;

  return (
    <div className="event-log panel">
      <div className="event-log-header">
        <h4 className="section-label" style={{ margin: 0 }}>Event Log</h4>
        <span className="event-log-count">{entries.length} events</span>
      </div>
      <div className="event-log-scroll" ref={scrollRef}>
        {entries.map((e, i) => (
          <div key={i} className={`event-line event-line--${e.level}`}>
            <span className="event-time">{e.time}</span>
            <span className="event-msg">{e.message}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
