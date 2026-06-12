import React, { useEffect, useState } from 'react';
import { Megaphone } from 'lucide-react';
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
    <div className="bg-primary text-primary-foreground flex items-center overflow-hidden h-10 border-b border-primary-foreground/10 sticky top-0 z-50">
      <div className="bg-primary-foreground/10 px-4 h-full flex items-center z-10 font-bold whitespace-nowrap shadow-md">
        <Megaphone className="w-4 h-4 mr-2" />
        آخر الأخبار
      </div>
      
      <div className="flex-1 overflow-hidden relative h-full flex items-center">
        {/* CSS Marquee Animation */}
        <div 
          className="whitespace-nowrap inline-block font-medium px-4"
          style={{
            animation: 'marquee 25s linear infinite',
            // To ensure animation plays even if content is short
            minWidth: '100%'
          }}
        >
          <span className="opacity-70 mr-2">[{ticker.priority.toUpperCase()}]</span>
          <span className="font-bold mr-2">{ticker.title}</span> - {ticker.message}
        </div>
      </div>

      <style>{`
        @keyframes marquee {
          0% { transform: translateX(100%); }
          100% { transform: translateX(-100%); }
        }
      `}</style>
    </div>
  );
};
