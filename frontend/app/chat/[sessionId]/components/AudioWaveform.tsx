import { memo } from 'react';
import { cn } from '@/app/lib/utils';

interface AudioWaveformProps {
  className?: string;
}

const BAR_COUNT = 28;

const bars = Array.from({ length: BAR_COUNT }, (_, i) => {
  const center = (BAR_COUNT - 1) / 2;
  const dist = Math.abs(i - center) / center;
  const maxScale = 1 - dist * 0.6;
  return { index: i, maxScale };
});

export const AudioWaveform = memo(function AudioWaveform({ className }: AudioWaveformProps) {
  return (
    <div className={cn('flex items-center gap-[2px] h-6', className)}>
      {bars.map(({ index, maxScale }) => (
        <span
          key={index}
          className="inline-block w-[3px] rounded-full bg-current animate-waveform"
          style={{
            animationDelay: `${index * 60}ms`,
            // @ts-expect-error CSS custom property
            '--waveform-max-scale': maxScale,
          }}
        />
      ))}
    </div>
  );
});
