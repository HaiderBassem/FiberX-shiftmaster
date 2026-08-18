import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { AlertCircle, AlertTriangle, Bell, Info, X } from 'lucide-react';
import { announcementService } from '../../services/announcementService';
import { assetUrl } from '@/lib/assets';

const getImageUrl = (url: string) => assetUrl(url);

export const AnnouncementBanner: React.FC = () => {
  const { t } = useTranslation();
  const [isVisible, setIsVisible] = useState(true);
  const [lightboxImg, setLightboxImg] = useState<string | null>(null);

  // Under the ['announcements'] prefix so the WebSocket announcement signal
  // refreshes this banner live; a new announcement used to require a reload.
  const { data: announcement, isPending: loading } = useQuery({
    queryKey: ['announcements', 'active'],
    queryFn: () => announcementService.getActive(),
    refetchInterval: 5 * 60 * 1000,
  });

  if (loading || !announcement || !isVisible) return null;

  const getPriorityStyles = (priority: string) => {
    switch (priority) {
      case 'critical':
        return 'bg-red-50 text-red-800 border-red-200 dark:bg-red-950/40 dark:text-red-200 dark:border-red-900';
      case 'important':
        return 'bg-amber-50 text-amber-800 border-amber-200 dark:bg-amber-950/40 dark:text-amber-200 dark:border-amber-900';
      case 'normal':
        return 'bg-blue-50 text-blue-800 border-blue-200 dark:bg-blue-950/40 dark:text-blue-200 dark:border-blue-900';
      case 'info':
      default:
        return 'bg-gray-50 text-gray-800 border-gray-200 dark:bg-gray-900/40 dark:text-gray-200 dark:border-gray-700';
    }
  };

  const getPriorityLabel = (priority: string) => {
    switch (priority) {
      case 'critical':
        return t('announcements.priority_critical');
      case 'important':
        return t('announcements.priority_important');
      case 'normal':
        return t('announcements.priority_normal');
      case 'info':
      default:
        return t('announcements.priority_info');
    }
  };

  const getPriorityIcon = (priority: string) => {
    switch (priority) {
      case 'critical':
        return <AlertTriangle className="h-5 w-5 text-red-600 dark:text-red-400" />;
      case 'important':
        return <AlertCircle className="h-5 w-5 text-amber-600 dark:text-amber-400" />;
      case 'normal':
        return <Bell className="h-5 w-5 text-blue-600 dark:text-blue-400" />;
      case 'info':
      default:
        return <Info className="h-5 w-5 text-gray-600 dark:text-gray-400" />;
    }
  };

  return (
    <>
      <div className={`mb-6 rounded-lg border p-4 shadow-sm relative overflow-hidden transition-all duration-300 ${getPriorityStyles(announcement.priority)}`}>
        <div className="flex items-start">
          <div className="flex-shrink-0 mt-0.5">
            {getPriorityIcon(announcement.priority)}
          </div>
          <div className="ms-3 flex-1">
            <h3 className="text-sm font-medium uppercase tracking-wider mb-1 opacity-80">
              {getPriorityLabel(announcement.priority)}
            </h3>
            <h4 className="text-lg font-semibold mb-2">{announcement.title}</h4>
            <div className="text-sm opacity-90 whitespace-pre-wrap">
              {announcement.message}
            </div>

            {/* Images */}
            {announcement.images && announcement.images.length > 0 && (
              <div className="flex gap-2 mt-3 overflow-x-auto pb-1">
                {announcement.images.map((img, idx) => (
                  <button
                    key={idx}
                    onClick={() => setLightboxImg(getImageUrl(img))}
                    className="relative flex-shrink-0 group rounded-lg overflow-hidden border border-black/10 hover:border-black/30 dark:border-white/15 dark:hover:border-white/40 transition-colors"
                  >
                    <img
                      src={getImageUrl(img)}
                      alt={`Attachment ${idx + 1}`}
                      className="w-16 h-16 sm:w-20 sm:h-20 object-cover transition-transform group-hover:scale-105"
                    />
                  </button>
                ))}
              </div>
            )}

            <div className="mt-3 text-xs opacity-70">
              {t('announcements.posted_by', {
                name: announcement.creator_name || t('announcements.management'),
                date: new Date(announcement.created_at).toLocaleDateString(),
              })}
            </div>
          </div>
          <button
            onClick={() => setIsVisible(false)}
            className="ms-auto -mx-1.5 -my-1.5 bg-transparent p-1.5 rounded-lg inline-flex h-8 w-8 hover:bg-black/5 dark:hover:bg-white/10 focus:ring-2 focus:ring-black/10 dark:focus:ring-white/20"
          >
            <span className="sr-only">{t('announcements.dismiss')}</span>
            <X className="h-5 w-5" />
          </button>
        </div>
        
        {/* Decorative accent bar */}
        <div className={`absolute top-0 start-0 w-1 h-full ${
          announcement.priority === 'critical' ? 'bg-red-500' :
          announcement.priority === 'important' ? 'bg-amber-500' :
          announcement.priority === 'normal' ? 'bg-blue-500' :
          'bg-gray-500'
        }`} />
      </div>

      {/* Lightbox */}
      {lightboxImg && (
        <div
          className="fixed inset-0 z-[60] flex items-center justify-center bg-black/80 p-4 animate-in fade-in duration-200"
          onClick={() => setLightboxImg(null)}
        >
          <button
            className="absolute top-4 right-4 w-10 h-10 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center text-white transition-colors z-10"
            onClick={() => setLightboxImg(null)}
          >
            <X className="w-5 h-5" />
          </button>
          <img
            src={lightboxImg}
            alt="Full size"
            className="max-w-full max-h-[85vh] rounded-xl shadow-2xl object-contain animate-in zoom-in-95 duration-200"
            onClick={(e) => e.stopPropagation()}
          />
        </div>
      )}
    </>
  );
};
