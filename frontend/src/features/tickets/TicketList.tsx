import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { MessageSquare, Plus, CheckCircle2, Ticket as TicketIcon, Image as ImageIcon, X } from 'lucide-react';
import { format } from 'date-fns';
import { ar, enUS } from 'date-fns/locale';
import { useTranslation } from 'react-i18next';
import api from '@/lib/api';
import { toast } from 'sonner';
import { useAuthStore } from '@/store/authStore';

export const TicketList = () => {
  const { t, i18n } = useTranslation();
  const queryClient = useQueryClient();
  const { user } = useAuthStore();
  const dateLocale = i18n.language === 'ar' ? ar : enUS;

  const [isCreating, setIsCreating] = useState(false);
  const [selectedTicketId, setSelectedTicketId] = useState<string | null>(null);

  // Form State
  const [targetDeptId, setTargetDeptId] = useState('');
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [images, setImages] = useState<string[]>([]);
  const [isUploading, setIsUploading] = useState(false);

  // Fetch Departments
  const { data: departments = [] } = useQuery({
    queryKey: ['departments'],
    queryFn: async () => {
      const res = await api.get('/departments');
      return res.data?.data || [];
    }
  });

  // Fetch Tickets
  const { data: tickets = [], isLoading } = useQuery({
    queryKey: ['tickets'],
    queryFn: async () => {
      const res = await api.get('/tickets');
      return res.data?.data || [];
    },
    refetchInterval: 15000,
  });

  const uploadImage = async (file: File) => {
    setIsUploading(true);
    const formData = new FormData();
    formData.append('image', file);
    try {
      const res = await api.post('/upload/image', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      if (res.data?.data?.url) {
        setImages((prev) => [...prev, res.data.data.url]);
      } else if (res.data?.file) {
        setImages((prev) => [...prev, res.data.file]);
      }
    } catch (err) {
      toast.error(t('common.failed_upload'));
    } finally {
      setIsUploading(false);
    }
  };

  const createTicket = useMutation({
    mutationFn: async () => {
      await api.post('/tickets', {
        target_department_id: targetDeptId,
        title,
        description,
        attachments: images.length > 0 ? JSON.stringify(images) : null,
      });
    },
    onSuccess: () => {
      toast.success(t('common.success'));
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      setIsCreating(false);
      setTargetDeptId('');
      setTitle('');
      setDescription('');
      setImages([]);
    },
    onError: () => toast.error(t('tickets.failed_create')),
  });

  const closeTicket = useMutation({
    mutationFn: async (id: string) => {
      await api.put(`/tickets/${id}/close`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      toast.success(t('common.success'));
    },
    onError: () => toast.error(t('tickets.failed_close')),
  });

  const handleCreateSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim() || !description.trim() || !targetDeptId) return;
    createTicket.mutate();
  };

  if (isLoading) return <div className="p-8 text-center text-muted-foreground">Loading...</div>;

  return (
    <div className="max-w-4xl mx-auto space-y-6 pb-20">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <TicketIcon className="w-6 h-6 text-primary" /> {t('tickets.title')}
          </h1>
          <p className="text-muted-foreground">{t('tickets.desc')}</p>
        </div>
        {!isCreating && (
          <Button onClick={() => setIsCreating(true)} className="rounded-xl shadow-lg shadow-primary/20">
            <Plus className="w-4 h-4 mr-2" /> {t('tickets.create')}
          </Button>
        )}
      </div>

      {isCreating && (
        <Card className="bg-gradient-to-br from-background to-muted/20 border-white/5 shadow-xl">
          <CardContent className="p-6">
            <h2 className="text-xl font-bold mb-4">{t('tickets.create_ticket')}</h2>
            <form onSubmit={handleCreateSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label>{t('tickets.target_department')}</Label>
                <select
                  className="w-full h-11 px-4 rounded-xl bg-muted/50 border-transparent focus:border-primary focus:bg-background transition-colors"
                  value={targetDeptId}
                  onChange={(e) => setTargetDeptId(e.target.value)}
                  required
                >
                  <option value="">{t('tickets.select_department')}</option>
                  {(departments || [])
                    .filter((d: any) => d.id !== user?.department_id)
                    .map((d: any) => (
                      <option key={d.id} value={d.id}>{d.name}</option>
                  ))}
                </select>
              </div>

              <div className="space-y-2">
                <Label>{t('tickets.ticket_title')}</Label>
                <Input
                  required
                  placeholder={t('tickets.title_placeholder')}
                  value={title}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setTitle(e.target.value)}
                  className="bg-muted/50 border-transparent"
                />
              </div>

              <div className="space-y-2">
                <Label>{t('tickets.description')}</Label>
                <textarea
                  required
                  placeholder={t('tickets.desc_placeholder')}
                  value={description}
                  onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setDescription(e.target.value)}
                  className="w-full min-h-[100px] px-4 py-3 rounded-xl bg-muted/50 border-transparent focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary/20 transition-all"
                />
              </div>

              {/* Image Upload */}
              <div className="space-y-2">
                <Label className="flex items-center gap-2 cursor-pointer text-sm text-primary hover:text-primary/80 transition-colors w-fit">
                  <input
                    type="file"
                    accept="image/*"
                    className="hidden"
                    onChange={(e) => {
                      if (e.target.files?.[0]) uploadImage(e.target.files[0]);
                      e.target.value = '';
                    }}
                    disabled={isUploading}
                  />
                  <ImageIcon className="w-4 h-4" />
                  {isUploading ? 'Uploading...' : 'Attach Image'}
                </Label>
                {images.length > 0 && (
                  <div className="flex gap-2 mt-2">
                    {images.map((img, i) => (
                      <div key={i} className="relative w-16 h-16">
                        <img src={img} alt="attachment" className="w-full h-full object-cover rounded-lg border border-border" />
                        <button
                          type="button"
                          onClick={() => setImages(images.filter((_, idx) => idx !== i))}
                          className="absolute -top-2 -right-2 bg-destructive text-white rounded-full p-0.5 shadow-lg"
                        >
                          <X className="w-3 h-3" />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="flex justify-end gap-3 pt-4">
                <Button type="button" variant="ghost" onClick={() => setIsCreating(false)}>
                  {t('common.cancel', 'Cancel')}
                </Button>
                <Button type="submit" disabled={createTicket.isPending || isUploading}>
                  {t('tickets.submit')}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      {/* Ticket List */}
      <div className="space-y-4">
        {tickets.length === 0 && !isCreating && (
          <div className="text-center py-12 px-4 rounded-3xl border border-dashed border-border/50 bg-muted/10">
            <TicketIcon className="w-12 h-12 text-muted-foreground/30 mx-auto mb-3" />
            <p className="text-muted-foreground">{t('tickets.no_tickets')}</p>
          </div>
        )}

        {tickets.map((ticket: any) => {
          const isExpanded = selectedTicketId === ticket.id;
          const isClosed = ticket.status === 'closed';

          return (
            <Card
              key={ticket.id}
              className={`border-white/5 transition-all overflow-hidden ${isClosed ? 'opacity-75 bg-muted/10' : 'bg-background hover:shadow-lg'}`}
            >
              <div 
                className="p-5 flex flex-col sm:flex-row sm:items-start justify-between gap-4 cursor-pointer"
                onClick={() => setSelectedTicketId(isExpanded ? null : ticket.id)}
              >
                <div className="flex items-start gap-4 flex-1">
                  <div className={`w-10 h-10 rounded-2xl flex items-center justify-center shrink-0 ${isClosed ? 'bg-muted text-muted-foreground' : 'bg-primary/10 text-primary'}`}>
                    <TicketIcon className="w-5 h-5" />
                  </div>
                  <div>
                    <h3 className="font-bold text-lg">{ticket.title}</h3>
                    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted-foreground mt-1">
                      <span>{t('tickets.ticket_from')} <b>{ticket.source_department}</b></span>
                      <span>{t('tickets.to')} <b>{ticket.target_department}</b></span>
                      <span className="flex items-center gap-1 opacity-70">
                        {format(new Date(ticket.created_at), 'MMM d, p', { locale: dateLocale })}
                      </span>
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-3 shrink-0">
                  <span className={`px-3 py-1 rounded-full text-xs font-semibold ${isClosed ? 'bg-muted text-muted-foreground' : 'bg-green-500/10 text-green-500'}`}>
                    {isClosed ? t('tickets.status_closed') : t('tickets.status_open')}
                  </span>
                  {!isClosed && (
                    <Button
                      variant="outline"
                      size="sm"
                      className="gap-2 h-8 text-xs hover:bg-green-500 hover:text-white hover:border-green-500"
                      onClick={(e: React.MouseEvent) => {
                        e.stopPropagation();
                        if (window.confirm('Close this ticket?')) {
                          closeTicket.mutate(ticket.id);
                        }
                      }}
                    >
                      <CheckCircle2 className="w-3.5 h-3.5" />
                      {t('tickets.mark_as_closed')}
                    </Button>
                  )}
                </div>
              </div>

              {isExpanded && (
                <div className="px-5 pb-5 pt-2 border-t border-white/5 bg-muted/5">
                  <div className="py-4">
                    <p className="text-foreground whitespace-pre-wrap">{ticket.description}</p>
                    
                    {ticket.attachments && JSON.parse(ticket.attachments).length > 0 && (
                      <div className="flex gap-3 mt-4 overflow-x-auto pb-2">
                        {JSON.parse(ticket.attachments).map((url: string, i: number) => (
                          <a key={i} href={url} target="_blank" rel="noreferrer" className="shrink-0">
                            <img src={url} alt="Attachment" className="h-24 w-24 object-cover rounded-xl border border-white/10 shadow-sm hover:opacity-80 transition-opacity" />
                          </a>
                        ))}
                      </div>
                    )}

                    {isClosed && ticket.closed_by_name && (
                      <div className="mt-4 inline-flex items-center gap-2 px-3 py-1.5 rounded-lg bg-muted text-sm text-muted-foreground border border-border">
                        <CheckCircle2 className="w-4 h-4 text-green-500" />
                        {t('tickets.closed_by')}: {ticket.closed_by_name}
                      </div>
                    )}
                  </div>

                  <TicketComments ticket={ticket} />
                </div>
              )}
            </Card>
          );
        })}
      </div>
    </div>
  );
};

// Extracted outside to prevent remounting on every parent render
const TicketComments = ({ ticket }: { ticket: any }) => {
  const { t, i18n } = useTranslation();
  const dateLocale = i18n.language === 'ar' ? ar : enUS;
  const queryClient = useQueryClient();

  const [commentText, setCommentText] = useState('');
  const [commentImages, setCommentImages] = useState<string[]>([]);
  const [uploading, setUploading] = useState(false);

  const uploadCommentImage = async (file: File) => {
    setUploading(true);
    const formData = new FormData();
    formData.append('image', file);
    try {
      const res = await api.post('/upload/image', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      if (res.data?.data?.url) {
        setCommentImages((prev) => [...prev, res.data.data.url]);
      } else if (res.data?.file) {
        setCommentImages((prev) => [...prev, res.data.file]);
      }
    } catch (err) {
      toast.error('Upload failed');
    } finally {
      setUploading(false);
    }
  };

  const addComment = useMutation({
    mutationFn: async () => {
      await api.post(`/tickets/${ticket.id}/comments`, {
        comment: commentText,
        attachments: commentImages.length > 0 ? JSON.stringify(commentImages) : null,
      });
    },
    onSuccess: () => {
      setCommentText('');
      setCommentImages([]);
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
    }
  });

  return (
    <div className="mt-6 pt-6 border-t border-border">
      <h4 className="text-sm font-semibold mb-4 text-muted-foreground flex items-center gap-2">
        <MessageSquare className="w-4 h-4" />
        {t('tickets.add_comment', 'Add Comment')}
      </h4>

      {/* Existing Comments */}
      <div className="space-y-4 mb-4">
        {(ticket.comments || []).map((c: any) => (
          <div key={c.id} className="bg-muted/30 p-3 rounded-xl">
            <div className="flex items-center gap-2 mb-2">
              <div className="w-6 h-6 rounded-full bg-primary/20 flex items-center justify-center text-xs font-bold">
                {c.author_name.charAt(0)}
              </div>
              <span className="text-sm font-medium">{c.author_name}</span>
              <span className="text-xs text-muted-foreground ml-auto">
                {format(new Date(c.created_at), 'PPp', { locale: dateLocale })}
              </span>
            </div>
            <p className="text-sm text-foreground whitespace-pre-wrap pl-8">{c.comment}</p>
            {c.attachments && JSON.parse(c.attachments).length > 0 && (
              <div className="flex gap-2 mt-2 pl-8 overflow-x-auto">
                {JSON.parse(c.attachments).map((url: string, i: number) => (
                  <a key={i} href={url} target="_blank" rel="noreferrer">
                    <img src={url} alt="Attachment" className="h-16 w-16 object-cover rounded-lg border border-white/10 hover:opacity-80 transition-opacity" />
                  </a>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Add Comment */}
      {ticket.status === 'open' && (
        <div className="flex gap-2 items-start mt-4">
          <div className="flex-1 space-y-2">
            <textarea
              placeholder={t('tickets.type_comment')}
              value={commentText}
              onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setCommentText(e.target.value)}
              className="w-full min-h-[44px] px-3 py-2 rounded-xl bg-muted/50 border-transparent focus:border-primary/50 focus:outline-none focus:ring-1 focus:ring-primary/20 resize-none transition-all"
            />
            {commentImages.length > 0 && (
              <div className="flex gap-2">
                {commentImages.map((img, i) => (
                  <div key={i} className="relative w-12 h-12">
                    <img src={img} alt="preview" className="w-full h-full object-cover rounded-lg border border-border" />
                    <button onClick={() => setCommentImages(commentImages.filter((_, idx) => idx !== i))} className="absolute -top-2 -right-2 bg-destructive text-white rounded-full p-0.5">
                      <X className="w-3 h-3" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="flex flex-col gap-2 shrink-0">
            <Label className="cursor-pointer h-11 px-3 bg-muted/50 hover:bg-muted text-muted-foreground rounded-xl border border-transparent flex items-center justify-center transition-colors">
              <input
                type="file"
                accept="image/*"
                className="hidden"
                onChange={(e) => {
                  if (e.target.files?.[0]) uploadCommentImage(e.target.files[0]);
                  e.target.value = '';
                }}
                disabled={uploading}
              />
              <ImageIcon className="w-5 h-5" />
            </Label>
            <Button
              size="icon"
              className="h-11 w-11 rounded-xl shrink-0"
              onClick={() => addComment.mutate()}
              disabled={!commentText.trim() || addComment.isPending}
            >
              <Plus className="w-5 h-5" />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
};

