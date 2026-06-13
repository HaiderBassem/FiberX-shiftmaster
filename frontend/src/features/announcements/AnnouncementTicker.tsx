import React, { useEffect, useState } from 'react';
import { announcementService } from '../../services/announcementService';
import type { Announcement } from '../../services/announcementService';

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

    // Optionally refresh every 5 mins
    const interval = setInterval(fetchTicker, 5 * 60 * 1000);
    return () => clearInterval(interval);
  }, []);

  if (!ticker) return null;

  return (
    <div className="bg-primary text-primary-foreground flex items-center overflow-hidden h-10 border-b border-primary-foreground/10 sticky top-0 z-50 w-full">
      <div className="flex-1 overflow-hidden relative h-full flex items-center min-w-0">
        {/* CSS Marquee Animation using absolute positioning to prevent layout stretching */}
        <div 
          className="whitespace-nowrap absolute font-medium px-4 flex items-center"
          style={{
            animation: 'marquee 30s linear infinite',
            willChange: 'transform'
          }}
        >
          <span className="opacity-70 mr-2">[{ticker.priority.toUpperCase()}]</span>
          <span className="font-bold mr-2">{ticker.title}</span> - {ticker.message}
        </div>
      </div>

      <style>{`
        @keyframes marquee {
          0% { transform: translateX(100vw); }
          100% { transform: translateX(-100%); }
        }
      `}</style>
    </div>
  );
};
