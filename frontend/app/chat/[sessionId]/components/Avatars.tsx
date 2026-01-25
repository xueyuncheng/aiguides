import { memo } from 'react';
import { Avatar, AvatarFallback, AvatarImage } from '@/app/components/ui/avatar';

// AI Avatar component
export const AIAvatar = memo(({ icon }: { icon: string }) => {
  return (
    <div className="h-8 w-8 rounded-full flex items-center justify-center flex-shrink-0 border border-border bg-background shadow-sm">
      <span className="text-base">{icon}</span>
    </div>
  );
});

AIAvatar.displayName = 'AIAvatar';

// User Avatar component
export const UserAvatar = memo(({ user }: { user: { name: string; picture?: string } | null }) => {
  if (!user) return null;

  return (
    <Avatar className="h-8 w-8 flex-shrink-0">
      <AvatarImage src={user.picture} alt={user.name} />
      <AvatarFallback className="bg-blue-500 text-white text-sm">
        {user.name.charAt(0).toUpperCase()}
      </AvatarFallback>
    </Avatar>
  );
});

UserAvatar.displayName = 'UserAvatar';
