import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Plus, CheckCircle, Hand } from 'lucide-react';
import { useAuthStore } from '@/store/authStore';
import CreateHandoverModal from './CreateHandoverModal';
import { useTranslation } from 'react-i18next';

export type HandoverComment = {
  id: string;
  employee_id: string;
  author_name: string;
  comment: string;
  created_at: string;
};

export type Handover = {
  id: string;
  department_id: string;
  creator_id: string;
  shift_summary: string;
  pending_issues: string;
  status: 'open' | 'claimed' | 'completed';
  claimed_by: string | null;
  created_at: string;
  updated_at: string;
  creator_name?: string;
  claimer_name?: string;
  done_by_name?: string;
  comments?: HandoverComment[];
};

export default function HandoverBoard() {
  const { t } = useTranslation();
  const { user } = useAuthStore();
  const queryClient = useQueryClient();
  const [isCreateOpen, setIsCreateOpen] = useState(false);

  const { data: handovers, isLoading } = useQuery({
    queryKey: ['handovers'],
    queryFn: async () => {
      const res = await api.get('/handovers');
      return (res.data?.data || []) as Handover[];
    },
  });

  const claimMutation = useMutation({
    mutationFn: async (id: string) => api.put(`/handovers/${id}/claim`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['handovers'] }),
  });

  const completeMutation = useMutation({
    mutationFn: async (id: string) => api.put(`/handovers/${id}/complete`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['handovers'] }),
  });

  const unclaimMutation = useMutation({
    mutationFn: async (id: string) => api.put(`/handovers/${id}/unclaim`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['handovers'] }),
  });

  const [commentingOnId, setCommentingOnId] = useState<string | null>(null);
  const [commentText, setCommentText] = useState('');

  const addCommentMutation = useMutation({
    mutationFn: async ({ id, comment }: { id: string; comment: string }) => 
      api.post(`/handovers/${id}/comments`, { comment }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['handovers'] });
      setCommentingOnId(null);
      setCommentText('');
    },
  });

  const activeHandovers = handovers?.filter((h) => h.status !== 'completed') || [];
  const completedHandovers = handovers?.filter((h) => h.status === 'completed') || [];

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-foreground">{t('handovers.shift_handovers')}</h1>
          <p className="text-muted-foreground mt-1">{t('handovers.manage_handovers')}</p>
        </div>
        <button 
          onClick={() => setIsCreateOpen(true)}
          className="flex items-center px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
        >
          <Plus className="w-4 h-4 mr-2" />
          {t('handovers.create_handover')}
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="space-y-4">
          <h2 className="text-xl font-semibold flex items-center gap-2 text-foreground">
            {t('handovers.active_handovers')}
            <span className="bg-muted text-muted-foreground px-2 py-0.5 rounded-full text-xs">{activeHandovers.length}</span>
          </h2>
          {isLoading ? (
            <p>{t('common.loading')}</p>
          ) : activeHandovers.length === 0 ? (
            <p className="text-muted-foreground">{t('handovers.no_active_handovers')}</p>
          ) : (
            activeHandovers.map((h) => (
              <div key={h.id} className={`bg-card rounded-xl shadow-sm border ${h.status === 'open' ? 'border-l-4 border-l-yellow-500 border-border' : 'border-l-4 border-l-blue-500 border-border'}`}>
                <div className="p-4 border-b border-border flex justify-between items-start">
                  <div>
                    <h3 className="text-lg font-semibold text-foreground">{t('handovers.handover_from')} {h.creator_name}</h3>
                    <p className="text-xs text-muted-foreground">{new Date(h.created_at).toLocaleString()}</p>
                  </div>
                  <span className={`px-2 py-1 text-xs rounded-md font-medium ${h.status === 'open' ? 'bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 border border-yellow-500/20' : 'bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20'}`}>
                    {t(`common.${h.status}`)}
                  </span>
                </div>
                <div className="p-4 space-y-4">
                  <div>
                    <h4 className="text-sm font-semibold text-muted-foreground mb-2">{t('handovers.shift_summary_colon')}</h4>
                    <div 
                      className="text-sm text-foreground jodit-content max-w-none prose prose-sm dark:prose-invert" 
                      dangerouslySetInnerHTML={{ __html: h.shift_summary }} 
                    />
                  </div>
                  {h.pending_issues && (
                    <div className="pt-2">
                      <h4 className="text-sm font-semibold text-orange-500 mb-2">{t('handovers.pending_issues_colon')}</h4>
                      <div 
                        className="text-sm text-foreground jodit-content max-w-none prose prose-sm dark:prose-invert" 
                        dangerouslySetInnerHTML={{ __html: h.pending_issues }} 
                      />
                    </div>
                  )}
                  {h.status === 'claimed' && h.claimer_name && (
                    <div className="bg-muted/50 p-2 rounded-md text-sm mt-2 text-foreground border border-border">
                      <span className="font-semibold">{t('handovers.claimed_by_colon')}</span> {h.claimer_name}
                    </div>
                  )}

                  {h.comments && h.comments.length > 0 && (
                    <div className="mt-4 space-y-3">
                      <h4 className="text-sm font-semibold text-foreground mb-1">{t('handovers.comments_colon')}</h4>
                      {h.comments.map((c) => (
                        <div key={c.id} className="p-3 bg-muted/30 border border-border rounded-md">
                          <div className="flex justify-between items-center mb-1">
                            <span className="text-sm font-medium text-foreground">{c.author_name}</span>
                            <span className="text-xs text-muted-foreground">{new Date(c.created_at).toLocaleString()}</span>
                          </div>
                          <div className="text-sm text-foreground whitespace-pre-wrap">
                            {c.comment}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}

                  {commentingOnId === h.id && (
                    <div className="mt-4 p-3 bg-muted/50 rounded-lg border border-border">
                      <textarea
                        className="w-full text-sm bg-background border border-input text-foreground rounded-md shadow-sm focus:ring-primary focus:border-primary"
                        rows={3}
                        placeholder={t('handovers.type_comment')}
                        value={commentText}
                        onChange={(e) => setCommentText(e.target.value)}
                      />
                      <div className="mt-2 flex justify-end gap-2">
                        <button
                          onClick={() => {
                            setCommentingOnId(null);
                            setCommentText('');
                          }}
                          className="px-3 py-1.5 text-sm text-muted-foreground hover:text-foreground"
                        >
                          {t('common.cancel')}
                        </button>
                        <button
                          onClick={() => addCommentMutation.mutate({ id: h.id, comment: commentText })}
                          disabled={!commentText.trim() || addCommentMutation.isPending}
                          className="px-3 py-1.5 text-sm bg-primary text-primary-foreground rounded hover:bg-primary/90 disabled:opacity-50"
                        >
                          {t('handovers.save_comment')}
                        </button>
                      </div>
                    </div>
                  )}
                </div>
                <div className="p-4 bg-muted/30 rounded-b-xl border-t border-border flex justify-end gap-2">
                  {h.status === 'open' && (
                    <button
                      onClick={() => claimMutation.mutate(h.id)}
                      disabled={claimMutation.isPending}
                      className="flex items-center px-3 py-1.5 border border-border text-foreground rounded-lg hover:bg-accent transition-colors text-sm"
                    >
                      <Hand className="w-4 h-4 mr-2" />
                      {t('handovers.claim_issues')}
                    </button>
                  )}
                  {h.status === 'claimed' && (
                    <>
                      {h.claimed_by === user?.id && (
                        <button
                          onClick={() => unclaimMutation.mutate(h.id)}
                          disabled={unclaimMutation.isPending}
                          className="flex items-center px-3 py-1.5 border border-destructive/50 text-destructive hover:bg-destructive/10 rounded-lg transition-colors text-sm font-medium"
                        >
                          {t('handovers.unclaim')}
                        </button>
                      )}
                      <button
                        onClick={() => {
                          setCommentingOnId(h.id);
                          setCommentText('');
                        }}
                        disabled={commentingOnId === h.id}
                        className="flex items-center px-3 py-1.5 border border-primary/50 text-primary hover:bg-primary/10 rounded-lg transition-colors text-sm font-medium"
                      >
                        {t('handovers.add_comment')}
                      </button>
                      <button
                        onClick={() => completeMutation.mutate(h.id)}
                        disabled={completeMutation.isPending}
                        className="flex items-center px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors text-sm font-medium"
                      >
                        <CheckCircle className="w-4 h-4 mr-2" />
                        {t('handovers.mark_as_done')}
                      </button>
                    </>
                  )}
                </div>
              </div>
            ))
          )}
        </div>

        <div className="space-y-4">
          <h2 className="text-xl font-semibold flex items-center gap-2 text-muted-foreground">
            {t('handovers.history_completed')}
            <span className="bg-muted text-muted-foreground px-2 py-0.5 rounded-full text-xs">{completedHandovers.length}</span>
          </h2>
          {completedHandovers.slice(0, 10).map((h) => (
            <div key={h.id} className="bg-card rounded-xl shadow-sm border border-border opacity-70">
              <div className="p-4 flex justify-between items-start">
                <div>
                  <h3 className="text-md font-semibold text-foreground">{t('handovers.handover_from')} {h.creator_name}</h3>
                  <p className="text-xs text-muted-foreground mt-1">
                    {h.claimer_name && h.claimer_name !== h.done_by_name ? (
                      <>{t('handovers.claimed_by')} {h.claimer_name} {t('handovers.and_done_by')} {h.done_by_name}</>
                    ) : (
                      <>{t('handovers.done_by')} {h.done_by_name || h.claimer_name || t('handovers.unknown')}</>
                    )} {t('handovers.on')} {new Date(h.updated_at).toLocaleString()}
                  </p>
                </div>
                <CheckCircle className="w-5 h-5 text-emerald-500" />
              </div>
              <div className="px-4 pb-4 space-y-3">
                <div 
                  className="text-sm text-muted-foreground jodit-content max-w-none prose prose-sm dark:prose-invert line-clamp-3 overflow-hidden" 
                  dangerouslySetInnerHTML={{ __html: h.shift_summary }} 
                />
                
                {h.comments && h.comments.length > 0 && (
                  <div className="mt-3 pt-3 border-t border-border/50">
                    <h4 className="text-xs font-semibold text-foreground mb-2">{t('handovers.comments_colon')}</h4>
                    <div className="space-y-2">
                      {h.comments.map((c) => (
                        <div key={c.id} className="text-sm">
                          <span className="font-medium text-foreground">{c.author_name}: </span>
                          <span className="text-muted-foreground">{c.comment}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>

      <CreateHandoverModal
        isOpen={isCreateOpen}
        onClose={() => setIsCreateOpen(false)}
      />
    </div>
  );
}
