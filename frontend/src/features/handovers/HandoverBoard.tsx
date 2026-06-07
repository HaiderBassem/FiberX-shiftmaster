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

  const activeHandovers = handovers?.filter((h) => h.status !== 'completed') || [];
  const completedHandovers = handovers?.filter((h) => h.status === 'completed') || [];

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">Shift Handovers</h1>
          <p className="text-gray-500 dark:text-gray-400 mt-1">Manage shift handovers and pending issues.</p>
        </div>
        <button 
          onClick={() => setIsCreateOpen(true)}
          className="flex items-center px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors"
        >
          <Plus className="w-4 h-4 mr-2" />
          Create Handover
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="space-y-4">
          <h2 className="text-xl font-semibold flex items-center gap-2 text-gray-900 dark:text-white">
            Active Handovers
            <span className="bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 px-2 py-0.5 rounded-full text-xs">{activeHandovers.length}</span>
          </h2>
          {isLoading ? (
            <p>Loading...</p>
          ) : activeHandovers.length === 0 ? (
            <p className="text-gray-500">No active handovers.</p>
          ) : (
            activeHandovers.map((h) => (
              <div key={h.id} className={`bg-white dark:bg-gray-800 rounded-xl shadow-sm border ${h.status === 'open' ? 'border-l-4 border-l-yellow-500 border-gray-200 dark:border-gray-700' : 'border-l-4 border-l-blue-500 border-gray-200 dark:border-gray-700'}`}>
                <div className="p-4 border-b border-gray-100 dark:border-gray-700 flex justify-between items-start">
                  <div>
                    <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Handover from {h.creator_name}</h3>
                    <p className="text-xs text-gray-500">{new Date(h.created_at).toLocaleString()}</p>
                  </div>
                  <span className={`px-2 py-1 text-xs rounded-md font-medium ${h.status === 'open' ? 'bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 border border-yellow-500/20' : 'bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20'}`}>
                    {h.status.toUpperCase()}
                  </span>
                </div>
                <div className="p-4 space-y-4">
                  <div>
                    <h4 className="text-sm font-semibold text-gray-500 dark:text-gray-400 mb-2">Shift Summary:</h4>
                    <div 
                      className="text-sm text-gray-800 dark:text-gray-200 jodit-content max-w-none prose prose-sm dark:prose-invert" 
                      dangerouslySetInnerHTML={{ __html: h.shift_summary }} 
                    />
                  </div>
                  {h.pending_issues && (
                    <div className="pt-2">
                      <h4 className="text-sm font-semibold text-orange-500 mb-2">Pending Issues:</h4>
                      <div 
                        className="text-sm text-gray-800 dark:text-gray-200 jodit-content max-w-none prose prose-sm dark:prose-invert" 
                        dangerouslySetInnerHTML={{ __html: h.pending_issues }} 
                      />
                    </div>
                  )}
                  {h.status === 'claimed' && (
                    <div className="bg-gray-50 dark:bg-gray-900/50 p-2 rounded-md text-sm mt-2 text-gray-700 dark:text-gray-300 border border-gray-100 dark:border-gray-700">
                      <span className="font-semibold">Claimed by:</span> {h.claimer_name}
                    </div>
                  )}
                </div>
                <div className="p-4 bg-gray-50 dark:bg-gray-800/50 rounded-b-xl border-t border-gray-100 dark:border-gray-700 flex justify-end gap-2">
                  {h.status === 'open' && (
                    <button
                      onClick={() => claimMutation.mutate(h.id)}
                      disabled={claimMutation.isPending}
                      className="flex items-center px-3 py-1.5 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors text-sm"
                    >
                      <Hand className="w-4 h-4 mr-2" />
                      Claim Issues
                    </button>
                  )}
                  {h.status === 'claimed' && h.claimed_by === user?.id && (
                    <button
                      onClick={() => completeMutation.mutate(h.id)}
                      disabled={completeMutation.isPending}
                      className="flex items-center px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors text-sm font-medium"
                    >
                      <CheckCircle className="w-4 h-4 mr-2" />
                      Mark as Done
                    </button>
                  )}
                </div>
              </div>
            ))
          )}
        </div>

        <div className="space-y-4">
          <h2 className="text-xl font-semibold flex items-center gap-2 text-gray-500 dark:text-gray-400">
            History (Completed)
            <span className="bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 px-2 py-0.5 rounded-full text-xs">{completedHandovers.length}</span>
          </h2>
          {completedHandovers.slice(0, 10).map((h) => (
            <div key={h.id} className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 opacity-70">
              <div className="p-4 flex justify-between items-start">
                <div>
                  <h3 className="text-md font-semibold text-gray-900 dark:text-white">Handover from {h.creator_name}</h3>
                  <p className="text-xs text-gray-500">
                    Completed by {h.claimer_name} on {new Date(h.updated_at).toLocaleString()}
                  </p>
                </div>
                <CheckCircle className="w-5 h-5 text-emerald-500" />
              </div>
              <div className="px-4 pb-4">
                <div 
                  className="text-sm text-gray-500 dark:text-gray-400 jodit-content max-w-none prose prose-sm dark:prose-invert line-clamp-3 overflow-hidden" 
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
