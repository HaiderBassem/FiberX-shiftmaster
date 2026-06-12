import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Plus, CheckCircle, Hand } from 'lucide-react';
import { useAuthStore } from '@/store/authStore';
import CreateHandoverModal from './CreateHandoverModal';

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
  claimer_notes?: string;
};

export default function HandoverBoard() {
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
          <h1 className="text-3xl font-bold tracking-tight text-foreground">Shift Handovers</h1>
          <p className="text-muted-foreground mt-1">Manage shift handovers and pending issues.</p>
        </div>
        <button 
          onClick={() => setIsCreateOpen(true)}
          className="flex items-center px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
        >
          <Plus className="w-4 h-4 mr-2" />
          Create Handover
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="space-y-4">
          <h2 className="text-xl font-semibold flex items-center gap-2 text-foreground">
            Active Handovers
            <span className="bg-muted text-muted-foreground px-2 py-0.5 rounded-full text-xs">{activeHandovers.length}</span>
          </h2>
          {isLoading ? (
            <p>Loading...</p>
          ) : activeHandovers.length === 0 ? (
            <p className="text-muted-foreground">No active handovers.</p>
          ) : (
            activeHandovers.map((h) => (
              <div key={h.id} className={`bg-card rounded-xl shadow-sm border ${h.status === 'open' ? 'border-l-4 border-l-yellow-500 border-border' : 'border-l-4 border-l-blue-500 border-border'}`}>
                <div className="p-4 border-b border-border flex justify-between items-start">
                  <div>
                    <h3 className="text-lg font-semibold text-foreground">Handover from {h.creator_name}</h3>
                    <p className="text-xs text-muted-foreground">{new Date(h.created_at).toLocaleString()}</p>
                  </div>
                  <span className={`px-2 py-1 text-xs rounded-md font-medium ${h.status === 'open' ? 'bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 border border-yellow-500/20' : 'bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20'}`}>
                    {h.status.toUpperCase()}
                  </span>
                </div>
                <div className="p-4 space-y-4">
                  <div>
                    <h4 className="text-sm font-semibold text-muted-foreground mb-2">Shift Summary:</h4>
                    <div 
                      className="text-sm text-foreground jodit-content max-w-none prose prose-sm dark:prose-invert" 
                      dangerouslySetInnerHTML={{ __html: h.shift_summary }} 
                    />
                  </div>
                  {h.pending_issues && (
                    <div className="pt-2">
                      <h4 className="text-sm font-semibold text-orange-500 mb-2">Pending Issues:</h4>
                      <div 
                        className="text-sm text-foreground jodit-content max-w-none prose prose-sm dark:prose-invert" 
                        dangerouslySetInnerHTML={{ __html: h.pending_issues }} 
                      />
                    </div>
                  )}
                  {h.status === 'claimed' && (
                    <div className="bg-muted/50 p-2 rounded-md text-sm mt-2 text-foreground border border-border">
                      <span className="font-semibold">Claimed by:</span> {h.claimer_name}
                    </div>
                  )}
                  {h.claimer_notes && (
                    <div className="mt-4 p-3 bg-blue-500/10 border border-blue-500/20 rounded-md">
                      <h4 className="text-sm font-semibold text-blue-600 dark:text-blue-400 mb-1">Claimer Notes:</h4>
                      <div className="text-sm text-blue-900 dark:text-blue-200 whitespace-pre-wrap">
                        {h.claimer_notes}
                      </div>
                    </div>
                  )}

                  {commentingOnId === h.id && (
                    <div className="mt-4 p-3 bg-muted/50 rounded-lg border border-border">
                      <textarea
                        className="w-full text-sm bg-background border border-input text-foreground rounded-md shadow-sm focus:ring-primary focus:border-primary"
                        rows={3}
                        placeholder="Type your comment/notes here..."
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
                          Cancel
                        </button>
                        <button
                          onClick={() => addCommentMutation.mutate({ id: h.id, comment: commentText })}
                          disabled={!commentText.trim() || addCommentMutation.isPending}
                          className="px-3 py-1.5 text-sm bg-primary text-primary-foreground rounded hover:bg-primary/90 disabled:opacity-50"
                        >
                          Save Comment
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
                      Claim Issues
                    </button>
                  )}
                  {h.status === 'claimed' && h.claimed_by === user?.id && (
                    <>
                      <button
                        onClick={() => unclaimMutation.mutate(h.id)}
                        disabled={unclaimMutation.isPending}
                        className="flex items-center px-3 py-1.5 border border-destructive/50 text-destructive hover:bg-destructive/10 rounded-lg transition-colors text-sm font-medium"
                      >
                        Unclaim
                      </button>
                      <button
                        onClick={() => {
                          setCommentingOnId(h.id);
                          setCommentText('');
                        }}
                        disabled={commentingOnId === h.id}
                        className="flex items-center px-3 py-1.5 border border-primary/50 text-primary hover:bg-primary/10 rounded-lg transition-colors text-sm font-medium"
                      >
                        Add Comment
                      </button>
                      <button
                        onClick={() => completeMutation.mutate(h.id)}
                        disabled={completeMutation.isPending}
                        className="flex items-center px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors text-sm font-medium"
                      >
                        <CheckCircle className="w-4 h-4 mr-2" />
                        Mark as Done
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
            History (Completed)
            <span className="bg-muted text-muted-foreground px-2 py-0.5 rounded-full text-xs">{completedHandovers.length}</span>
          </h2>
          {completedHandovers.slice(0, 10).map((h) => (
            <div key={h.id} className="bg-card rounded-xl shadow-sm border border-border opacity-70">
              <div className="p-4 flex justify-between items-start">
                <div>
                  <h3 className="text-md font-semibold text-foreground">Handover from {h.creator_name}</h3>
                  <p className="text-xs text-muted-foreground">
                    Completed by {h.claimer_name} on {new Date(h.updated_at).toLocaleString()}
                  </p>
                </div>
                <CheckCircle className="w-5 h-5 text-emerald-500" />
              </div>
              <div className="px-4 pb-4">
                <div 
                  className="text-sm text-muted-foreground jodit-content max-w-none prose prose-sm dark:prose-invert line-clamp-3 overflow-hidden" 
                  dangerouslySetInnerHTML={{ __html: h.shift_summary }} 
                />
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
