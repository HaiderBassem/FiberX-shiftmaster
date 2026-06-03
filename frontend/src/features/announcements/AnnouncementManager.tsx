import React, { useEffect, useState } from 'react';
import { Plus, Trash2, Power, AlertCircle, Loader, Shield } from 'lucide-react';
import { announcementService } from '../../services/announcementService';
import type { Announcement } from '../../services/announcementService';
import { AnnouncementPermissionsModal } from './AnnouncementPermissionsModal';
import { useAuthStore } from '@/store/authStore';

export const AnnouncementManager: React.FC = () => {
  const { user } = useAuthStore();
  const isAdminOrManager = user?.role === 'admin' || user?.role === 'manager';
  
  const [announcements, setAnnouncements] = useState<Announcement[]>([]);
  const [loading, setLoading] = useState(true);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isPermissionsOpen, setIsPermissionsOpen] = useState(false);
  
  const [title, setTitle] = useState('');
  const [message, setMessage] = useState('');
  const [priority, setPriority] = useState<'info' | 'normal' | 'important' | 'critical'>('normal');
  const [isActive, setIsActive] = useState(true);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    fetchAnnouncements();
  }, []);

  const fetchAnnouncements = async () => {
    try {
      setLoading(true);
      const data = await announcementService.getAll();
      setAnnouncements(data);
    } catch (err) {
      console.error('Failed to fetch announcements:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setSubmitting(true);
      await announcementService.create({
        title,
        message,
        priority,
        is_active: isActive,
      });
      setIsModalOpen(false);
      resetForm();
      fetchAnnouncements();
    } catch (err) {
      console.error('Failed to create announcement:', err);
      alert('Failed to create announcement.');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm('Are you sure you want to delete this announcement?')) return;
    try {
      await announcementService.delete(id);
      fetchAnnouncements();
    } catch (err) {
      console.error('Failed to delete announcement:', err);
      alert('Failed to delete announcement.');
    }
  };

  const handleToggleActive = async (id: string) => {
    try {
      await announcementService.setActive(id);
      fetchAnnouncements();
    } catch (err) {
      console.error('Failed to activate announcement:', err);
      alert('Failed to activate announcement.');
    }
  };

  const resetForm = () => {
    setTitle('');
    setMessage('');
    setPriority('normal');
    setIsActive(true);
  };

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Announcements</h2>
          <p className="text-muted-foreground">Manage department announcements</p>
        </div>
        <div className="flex gap-2">
          {isAdminOrManager && (
            <button
              onClick={() => setIsPermissionsOpen(true)}
              className="flex items-center space-x-2 rounded-lg border border-border bg-background px-4 py-2 text-sm font-medium text-foreground hover:bg-muted transition-colors shadow-sm"
            >
              <Shield className="h-4 w-4" />
              <span>Permissions</span>
            </button>
          )}
          <button
            onClick={() => setIsModalOpen(true)}
            className="flex items-center space-x-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors shadow-sm"
          >
            <Plus className="h-4 w-4" />
            <span>New Announcement</span>
          </button>
        </div>
      </div>

      <div className="rounded-lg border bg-card shadow-sm overflow-hidden">
        {announcements.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <AlertCircle className="h-12 w-12 text-muted-foreground/50 mb-4" />
            <h3 className="text-lg font-medium">No Announcements</h3>
            <p className="text-sm text-muted-foreground max-w-sm mt-2">
              There are no announcements to display. Create a new one to notify your department.
            </p>
          </div>
        ) : (
          <div className="divide-y divide-border">
            {announcements.map((a) => (
              <div key={a.id} className={`p-4 transition-colors ${a.is_active ? 'bg-primary/5 border-l-4 border-l-primary' : 'hover:bg-muted/50 border-l-4 border-l-transparent'}`}>
                <div className="flex items-start justify-between">
                  <div className="space-y-1">
                    <div className="flex items-center space-x-2">
                      <h4 className="font-semibold text-lg">{a.title}</h4>
                      {a.is_active && (
                        <span className="inline-flex items-center rounded-full bg-primary/10 px-2.5 py-0.5 text-xs font-medium text-primary">
                          Active
                        </span>
                      )}
                      <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium uppercase tracking-wider
                        ${a.priority === 'critical' ? 'bg-red-100 text-red-800' :
                          a.priority === 'important' ? 'bg-amber-100 text-amber-800' :
                          a.priority === 'info' ? 'bg-gray-100 text-gray-800' :
                          'bg-blue-100 text-blue-800'}`}>
                        {a.priority}
                      </span>
                    </div>
                    <p className="text-sm text-muted-foreground whitespace-pre-wrap max-w-4xl">{a.message}</p>
                    <div className="text-xs text-muted-foreground/80 mt-2">
                      Created by {a.creator_name || 'Management'} on {new Date(a.created_at).toLocaleString()}
                    </div>
                  </div>
                  
                  <div className="flex items-center space-x-2 ml-4">
                    {!a.is_active && (
                      <button
                        onClick={() => handleToggleActive(a.id)}
                        className="p-2 text-muted-foreground hover:text-primary transition-colors"
                        title="Set Active"
                      >
                        <Power className="h-5 w-5" />
                      </button>
                    )}
                    <button
                      onClick={() => handleDelete(a.id)}
                      className="p-2 text-muted-foreground hover:text-destructive transition-colors"
                      title="Delete"
                    >
                      <Trash2 className="h-5 w-5" />
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 animate-in fade-in duration-200">
          <div className="w-full max-w-md rounded-xl bg-background p-6 shadow-xl animate-in zoom-in-95 duration-200">
            <h3 className="mb-4 text-lg font-semibold">New Announcement</h3>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium">Title</label>
                <input
                  type="text"
                  required
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  className="w-full rounded-md border p-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  placeholder="e.g., Office Closure"
                />
              </div>

              <div>
                <label className="mb-1 block text-sm font-medium">Message</label>
                <textarea
                  required
                  value={message}
                  onChange={(e) => setMessage(e.target.value)}
                  rows={4}
                  className="w-full rounded-md border p-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary resize-none"
                  placeholder="Details of the announcement..."
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="mb-1 block text-sm font-medium">Priority</label>
                  <select
                    value={priority}
                    onChange={(e) => setPriority(e.target.value as any)}
                    className="w-full rounded-md border p-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  >
                    <option value="info">Info</option>
                    <option value="normal">Normal</option>
                    <option value="important">Important</option>
                    <option value="critical">Critical</option>
                  </select>
                </div>
                
                <div className="flex items-center mt-6">
                  <label className="flex items-center space-x-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={isActive}
                      onChange={(e) => setIsActive(e.target.checked)}
                      className="rounded border-gray-300 text-primary focus:ring-primary h-4 w-4"
                    />
                    <span className="text-sm font-medium">Set as Active</span>
                  </label>
                </div>
              </div>

              <div className="mt-2 text-xs text-muted-foreground bg-muted/50 p-2 rounded">
                Note: Setting this as Active will deactivate the currently active announcement (only 1 can be active). Active announcements will also send an email to all department members.
              </div>

              <div className="mt-6 flex justify-end space-x-3">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="rounded-md px-4 py-2 text-sm font-medium hover:bg-muted"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={submitting}
                  className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
                >
                  {submitting ? 'Creating...' : 'Create'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      <AnnouncementPermissionsModal
        isOpen={isPermissionsOpen}
        onClose={() => setIsPermissionsOpen(false)}
      />
    </div>
  );
};
