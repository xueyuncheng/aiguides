import { memo } from 'react';

export const ChatSkeleton = memo(() => {
  return (
    <div className="w-full max-w-5xl px-6 py-10 space-y-12 animate-skeleton">
      {[1, 2, 3].map((i) => (
        <div key={i} className="flex flex-col space-y-8">
          {/* User message skeleton */}
          <div className="flex justify-end">
            <div className="flex gap-4 max-w-[85%] flex-row-reverse items-start">
              <div className="h-8 w-8 rounded-full bg-secondary shrink-0" />
              <div className="bg-secondary/50 h-10 w-48 rounded-2xl rounded-tr-sm" />
            </div>
          </div>
          {/* Assistant message skeleton */}
          <div className="flex justify-start">
            <div className="flex gap-4 max-w-[85%] items-start">
              <div className="h-8 w-8 rounded-full bg-secondary shrink-0" />
              <div className="space-y-3 pt-1">
                <div className="h-4 bg-secondary/50 w-[300px] md:w-[500px] rounded" />
                <div className="h-4 bg-secondary/50 w-[200px] md:w-[400px] rounded" />
                <div className="h-4 bg-secondary/50 w-[250px] md:w-[450px] rounded" />
              </div>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
});

ChatSkeleton.displayName = 'ChatSkeleton';
