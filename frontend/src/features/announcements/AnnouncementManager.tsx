import React, { useEffect, useState, useRef } from 'react';
import { Plus, Trash2, Power, PowerOff, AlertCircle, Loader, Shield, ImagePlus, X } from 'lucide-react';
import { announcementService } from '../../services/announcementService';
import type { Announcement } from '../../services/announcementService';
import { AnnouncementPermissionsModal } from './AnnouncementPermissionsModal';
import { useAuthStore } from '@/store/authStore';

const getImageUrl = (url: string) => {
  if (url.startsWith('http')) return url;
  const base = import.meta.env.VITE_API_URL
    ? import.meta.env.VITE_API_URL.replace('/api', '')
    : (import.meta.env.DEV ? 'http://localhost:8080' : '');
  return `${base}${url.startsWith('/api') ? url : '/api' + url}`;
};

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
  const [isTicker, setIsTicker] = useState(false);
  const [imageFiles, setImageFiles] = useState<File[]>([]);
  const [imagePreviews, setImagePreviews] = useState<string[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Lightbox state
  const [lightboxImg, setLightboxImg] = useState<string | null>(null);

  useEffect(() => {
    fetchAnnouncements();
  }, []);

  // Generate previews when files change
  useEffect(() => {
    const urls = imageFiles.map(f => URL.createObjectURL(f));
    setImagePreviews(urls);
    return () => urls.forEach(u => URL.revokeObjectURL(u));
  }, [imageFiles]);

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

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || []);
    const total = imageFiles.length + files.length;
    if (total > 5) {
      alert('Maximum 5 images allowed per announcement.');
      return;
    }
    // Validate each file
    for (const f of files) {
      if (f.size > 5 * 1024 * 1024) {
        alert(`Image "${f.name}" exceeds 5MB limit.`);
        return;
      }
      if (!f.type.startsWith('image/')) {
        alert(`File "${f.name}" is not an image.`);
        return;
      }
    }
    setImageFiles(prev => [...prev, ...files]);
    // Reset input
    if (fileInputRef.current) fileInputRef.current.value = '';
  };

  const removeImage = (idx: number) => {
    setImageFiles(prev => prev.filter((_, i) => i !== idx));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setSubmitting(true);
      await announcementService.create(
        { title, message, priority, is_active: isActive, is_ticker: isTicker },
        imageFiles.length > 0 ? imageFiles : undefined
      );
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

  const handleToggleInactive = async (id: string) => {
    try {
      await announcementService.setInactive(id);
      fetchAnnouncements();
    } catch (err) {
      console.error('Failed to deactivate announcement:', err);
      alert('Failed to deactivate announcement.');
    }
  };

  const resetForm = () => {
    setTitle('');
    setMessage('');
    setPriority('normal');
    setIsActive(true);
    setIsTicker(false);
    setImageFiles([]);
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
      <div className="flex items-center justify-between flex-wrap gap-3">
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
                  <div className="space-y-1 flex-1 min-w-0">
                    <div className="flex items-center space-x-2 flex-wrap gap-1">
                      <h4 className="font-semibold text-lg">{a.title}</h4>
                      {a.is_active && (
                        <span className="inline-flex items-center rounded-full bg-primary/10 px-2.5 py-0.5 text-xs font-medium text-primary">
                          Active
                        </span>
                      )}
                      {a.is_ticker && (
                        <span className="inline-flex items-center rounded-full bg-indigo-100 text-indigo-800 px-2.5 py-0.5 text-xs font-medium border border-indigo-200">
                          Ticker
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
                    
                    {/* Display images */}
                    {a.images && a.images.length > 0 && (
                      <div className="flex flex-wrap gap-2 mt-3">
                        {a.images.map((img, idx) => (
                          <button
                            key={idx}
                            onClick={() => setLightboxImg(getImageUrl(img))}
                            className="relative group rounded-lg overflow-hidden border border-border hover:border-primary/50 transition-colors"
                          >
                            <img
                              src={getImageUrl(img)}
                              alt={`Attachment ${idx + 1}`}
                              className="w-20 h-20 object-cover transition-transform group-hover:scale-105"
                            />
                          </button>
                        ))}
                      </div>
                    )}

                    <div className="text-xs text-muted-foreground/80 mt-2">
                      Created by {a.creator_name || 'Management'} on {new Date(a.created_at).toLocaleString()}
                    </div>
                  </div>
                  
                  <div className="flex items-center space-x-2">
                      {!a.is_active ? (
                        <button
                          onClick={() => handleToggleActive(a.id)}
                          className="p-2 text-muted-foreground hover:text-primary transition-colors"
                          title="Set Active"
                        >
                          <Power className="h-5 w-5" />
                        </button>
                      ) : (
                        <button
                          onClick={() => handleToggleInactive(a.id)}
                          className="p-2 text-muted-foreground hover:text-amber-500 transition-colors"
                          title="Deactivate"
                        >
                          <PowerOff className="h-5 w-5" />
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

      {/* Create Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 animate-in fade-in duration-200">
          <div className="w-full max-w-lg rounded-xl bg-background p-6 shadow-xl animate-in zoom-in-95 duration-200 max-h-[90vh] overflow-y-auto">
            <h3 className="mb-4 text-lg font-semibold">New Announcement</h3>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium">Title</label>
                <input
                  type="text"
                  required
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  className="w-full rounded-md border p-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary bg-background"
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
                  className="w-full rounded-md border p-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary resize-none bg-background"
                  placeholder="Details of the announcement..."
                />
              </div>

              {/* Image Upload */}
              <div>
                <label className="mb-1.5 block text-sm font-medium">Images (optional)</label>
                <div className="space-y-3">
                  {/* Preview grid */}
                  {imagePreviews.length > 0 && (
                    <div className="flex flex-wrap gap-2">
                      {imagePreviews.map((url, idx) => (
                        <div key={idx} className="relative group">
                          <img
                            src={url}
                            alt={`Preview ${idx + 1}`}
                            className="w-20 h-20 object-cover rounded-lg border border-border"
                          />
                          <button
                            type="button"
                            onClick={() => removeImage(idx)}
                            className="absolute -top-1.5 -right-1.5 w-5 h-5 bg-destructive text-white rounded-full flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity shadow-sm"
                          >
                            <X className="w-3 h-3" />
                          </button>
                        </div>
                      ))}
                    </div>
                  )}

                  {/* Upload button */}
                  {imageFiles.length < 5 && (
                    <button
                      type="button"
                      onClick={() => fileInputRef.current?.click()}
                      className="flex items-center gap-2 px-3 py-2 rounded-lg border border-dashed border-border text-sm text-muted-foreground hover:text-foreground hover:border-primary/50 hover:bg-primary/5 transition-all"
                    >
                      <ImagePlus className="w-4 h-4" />
                      Add Images ({imageFiles.length}/5)
                    </button>
                  )}
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept="image/*"
                    multiple
                    onChange={handleFileSelect}
                    className="hidden"
                  />
                  <p className="text-xs text-muted-foreground">Max 5 images, 5MB each. Supports JPG, PNG, WebP.</p>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="mb-1 block text-sm font-medium">Priority</label>
                  <select
                    value={priority}
                    onChange={(e) => setPriority(e.target.value as any)}
                    className="w-full rounded-md border p-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary bg-background"
                  >
                    <option value="info">Info</option>
                    <option value="normal">Normal</option>
                    <option value="important">Important</option>
                    <option value="critical">Critical</option>
                  </select>
                </div>
                
                <div className="flex flex-col gap-3 mt-6">
                  <label className="flex items-center space-x-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={isActive}
                      onChange={(e) => setIsActive(e.target.checked)}
                      className="rounded border-gray-300 text-primary focus:ring-primary h-4 w-4"
                    />
                    <span className="text-sm font-medium">Set as Main Active (Banner)</span>
                  </label>

                  <label className="flex items-center space-x-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={isTicker}
                      onChange={(e) => setIsTicker(e.target.checked)}
                      className="rounded border-gray-300 text-primary focus:ring-primary h-4 w-4"
                    />
                    <span className="text-sm font-medium">Set as Ticker</span>
                  </label>
                </div>
              </div>

              <div className="mt-2 text-xs text-muted-foreground bg-muted/50 p-2 rounded">
                Note: Setting Main Active or Ticker will deactivate any currently active announcement of the same type. Active announcements send emails.
              </div>

              <div className="mt-6 flex justify-end space-x-3">
                <button
                  type="button"
                  onClick={() => { setIsModalOpen(false); resetForm(); }}
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

      {/* Image Lightbox */}
      {lightboxImg && (
        <div
          className="fixed inset-0 z-[60] flex items-center justify-center bg-black/80 p-4 animate-in fade-in duration-200"
          onClick={() => setLightboxImg(null)}
        >
          <button
            className="absolute top-4 right-4 w-10 h-10 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center text-white transition-colors"
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

      <AnnouncementPermissionsModal
        isOpen={isPermissionsOpen}
        onClose={() => setIsPermissionsOpen(false)}
      />
    </div>
  );
};
