import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { X } from 'lucide-react';

export default function CreateHandoverModal({
  isOpen,
  onClose,
}: {
  isOpen: boolean;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [shiftSummary, setShiftSummary] = useState('');
  const [pendingIssues, setPendingIssues] = useState('');
  const [error, setError] = useState('');

  const mutation = useMutation({
    mutationFn: async (data: any) => {
      const res = await api.post('/handovers', data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['handovers'] });
      setShiftSummary('');
      setPendingIssues('');
      setError('');
      onClose();
    },
    onError: (err: any) => {
      setError(err.response?.data?.error || 'Failed to create handover');
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!shiftSummary.trim()) {
      setError('Shift summary is required');
      return;
    }
    mutation.mutate({
      shift_summary: shiftSummary,
      pending_issues: pendingIssues,
    });
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-gray-900/50 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-2xl overflow-hidden">
        <div className="flex justify-between items-center p-6 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white">Create Shift Handover</h2>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200">
            <X className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="p-6 space-y-4">
            {error && <div className="p-3 bg-red-500/10 text-red-500 rounded-md text-sm">{error}</div>}

            <div className="space-y-2">
              <label htmlFor="shift_summary" className="block text-sm font-semibold text-gray-900 dark:text-white">Shift Summary <span className="text-red-500">*</span></label>
              <p className="text-xs text-gray-500 dark:text-gray-400 mb-1">What happened during your shift? Any major events?</p>
              <textarea
                id="shift_summary"
                value={shiftSummary}
                onChange={(e) => setShiftSummary(e.target.value)}
                placeholder="E.g., Smooth shift, all regular checks completed..."
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 min-h-[100px]"
              />
            </div>

            <div className="space-y-2">
              <label htmlFor="pending_issues" className="block text-sm font-semibold text-orange-500">Pending Issues</label>
              <p className="text-xs text-gray-500 dark:text-gray-400 mb-1">What is left for the next shift to handle?</p>
              <textarea
                id="pending_issues"
                value={pendingIssues}
                onChange={(e) => setPendingIssues(e.target.value)}
                placeholder="E.g., Ticket #1234 is still open, waiting for customer reply."
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 min-h-[100px]"
              />
            </div>
          </div>

          <div className="p-6 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 flex justify-end gap-3">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={mutation.isPending}
              className="px-6 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50 transition-colors shadow-sm font-medium"
            >
              {mutation.isPending ? 'Submitting...' : 'Submit Handover'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
