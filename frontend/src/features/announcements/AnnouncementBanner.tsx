import React, { useEffect, useState } from 'react';
import { AlertCircle, AlertTriangle, Bell, Info, X } from 'lucide-react';
import { announcementService } from '../../services/announcementService';
import type { Announcement } from '../../services/announcementService';

export const AnnouncementBanner: React.FC = () => {
  const [announcement, setAnnouncement] = useState<Announcement | null>(null);
  const [isVisible, setIsVisible] = useState(true);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchActiveAnnouncement();
  }, []);

  const fetchActiveAnnouncement = async () => {
    try {
      setLoading(true);
      const data = await announcementService.getActive();
      setAnnouncement(data);
    } catch (err) {
      console.error('Failed to fetch announcement:', err);
    } finally {
      setLoading(false);
    }
  };

  if (loading || !announcement || !isVisible) return null;

  const getPriorityStyles = (priority: string) => {
    switch (priority) {
      case 'critical':
        return 'bg-red-50 text-red-800 border-red-200';
      case 'important':
        return 'bg-amber-50 text-amber-800 border-amber-200';
      case 'normal':
        return 'bg-blue-50 text-blue-800 border-blue-200';
      case 'info':
      default:
        return 'bg-gray-50 text-gray-800 border-gray-200';
    }
  };

  const getPriorityIcon = (priority: string) => {
    switch (priority) {
      case 'critical':
        return <AlertTriangle className="h-5 w-5 text-red-600" />;
      case 'important':
        return <AlertCircle className="h-5 w-5 text-amber-600" />;
      case 'normal':
        return <Bell className="h-5 w-5 text-blue-600" />;
      case 'info':
      default:
        return <Info className="h-5 w-5 text-gray-600" />;
    }
  };

  return (
    <div className={`mb-6 rounded-lg border p-4 shadow-sm relative overflow-hidden transition-all duration-300 ${getPriorityStyles(announcement.priority)}`}>
      <div className="flex items-start">
        <div className="flex-shrink-0 mt-0.5">
          {getPriorityIcon(announcement.priority)}
        </div>
        <div className="ml-3 flex-1">
          <h3 className="text-sm font-medium uppercase tracking-wider mb-1 opacity-80">
            {announcement.priority} Announcement
          </h3>
          <h4 className="text-lg font-semibold mb-2">{announcement.title}</h4>
          <div className="text-sm opacity-90 whitespace-pre-wrap">
            {announcement.message}
          </div>
          <div className="mt-3 text-xs opacity-70">
            Posted by {announcement.creator_name || 'Management'} on {new Date(announcement.created_at).toLocaleDateString()}
          </div>
        </div>
        <button
          onClick={() => setIsVisible(false)}
          className="ml-auto -mx-1.5 -my-1.5 bg-transparent p-1.5 rounded-lg inline-flex h-8 w-8 hover:bg-black/5 focus:ring-2 focus:ring-black/10"
        >
          <span className="sr-only">Dismiss</span>
          <X className="h-5 w-5" />
        </button>
      </div>
      
      {/* Decorative accent bar */}
      <div className={`absolute top-0 left-0 w-1 h-full ${
        announcement.priority === 'critical' ? 'bg-red-500' :
        announcement.priority === 'important' ? 'bg-amber-500' :
        announcement.priority === 'normal' ? 'bg-blue-500' :
        'bg-gray-500'
      }`} />
    </div>
  );
};
