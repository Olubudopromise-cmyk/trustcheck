'use client';

export default function ErrorCard({ message }: { message: string }) {
  return (
    <div
      className="rounded-xl border border-red-200 bg-red-50 p-5 text-red-800 dark:border-red-900/40 dark:bg-red-950/40 dark:text-red-300"
      role="alert"
      aria-live="assertive"
    >
      <p className="font-medium">Verification could not be completed</p>
      <p className="mt-1">{message}</p>
      <p className="mt-2 text-xs text-red-700/80 dark:text-red-300/80">
        Your saved research has not been deleted. You can try again or reopen a previous session
        from the sidebar.
      </p>
    </div>
  );
}
