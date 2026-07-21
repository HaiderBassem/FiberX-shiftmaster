import React, { useEffect, useState } from 'react';
import { announcementService } from '../../services/announcementService';
import type { Announcement } from '../../services/announcementService';

/** Returns true if the string contains Arabic characters. */
function isArabic(text: string): boolean {
  return /[\u0600-\u06FF]/.test(text);
}

export const AnnouncementTicker: React.FC = () => {
  const [ticker, setTicker] = useState<Announcement | null>(null);

  useEffect(() => {
    const fetchTicker = async () => {
      try {
        const data = await announcementService.getActiveTicker();
        setTicker(data);
      } catch (err) {
        console.error('Failed to fetch ticker:', err);
      }
    };
    fetchTicker();

    // Refresh every 5 minutes
    const interval = setInterval(fetchTicker, 5 * 60 * 1000);
    return () => clearInterval(interval);
  }, []);

  if (!ticker) return null;

  // Detect direction from the announcement content (title takes priority)
  const fullText = `${ticker.title} ${ticker.message}`;
  const rtl = isArabic(fullText);

  // Scale animation duration with text length so speed feels consistent
  const duration = Math.max(20, Math.min(60, Math.round(fullText.length * 0.35)));
  const animationStyle: React.CSSProperties = {
    animationDuration: `${duration}s`,
  };

  return (
    <div className="bg-primary text-primary-foreground flex items-center overflow-hidden h-10 border-b border-primary-foreground/10 sticky top-0 z-10 w-full">
      <div className="flex-1 overflow-hidden relative h-full">
        <div
          dir={rtl ? 'rtl' : 'ltr'}
          style={animationStyle}
          className={`whitespace-nowrap absolute top-0 h-full flex items-center font-medium px-4 w-max ${
            rtl ? 'animate-marquee-rtl' : 'animate-marquee'
          }`}
        >
          <span className={`opacity-70 ${rtl ? 'ml-2' : 'mr-2'}`}>
            [{ticker.priority.toUpperCase()}]
          </span>
          <span className={`font-bold ${rtl ? 'ml-2' : 'mr-2'}`}>
            {ticker.title}
          </span>
          {' — '}
          {ticker.message}
        </div>
      </div>
    </div>
  );
};
