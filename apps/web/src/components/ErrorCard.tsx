'use client';

export default function ErrorCard({ message }: { message: string }) {
  return (
    <div
      className="rounded-xl border border-red-200 bg-red-50 p-5 text-red-800 dark:border-red-900/40 dark:bg-red-950/40 dark:text-red-300"
      role="alert"
      aria-live="assertive"
    >
      <p className="font-medium">Something went wrong</p>
      <p className="mt-1">{message}</p>
    </div>
  );
}
