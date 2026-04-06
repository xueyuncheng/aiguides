import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: '定时任务',
};

export default function ScheduledTasksLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
