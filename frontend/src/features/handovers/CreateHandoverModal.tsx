import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { X } from 'lucide-react';
import { RichTextEditor } from '@/components/RichTextEditor';

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
    <div className="fixed inset-0 z-50 overflow-y-auto bg-black/50 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-card rounded-xl shadow-xl w-full max-w-2xl overflow-hidden border border-border">
        <div className="flex justify-between items-center p-6 border-b border-border">
          <h2 className="text-xl font-semibold text-foreground">Create Shift Handover</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="p-6 space-y-4">
            {error && <div className="p-3 bg-red-500/10 text-red-500 rounded-md text-sm">{error}</div>}

            <div className="space-y-2">
              <label className="block text-sm font-semibold text-foreground">Shift Summary <span className="text-destructive">*</span></label>
              <p className="text-xs text-muted-foreground mb-1">What happened during your shift? Any major events?</p>
              <RichTextEditor
                value={shiftSummary}
                onChange={setShiftSummary}
                placeholder="E.g., Smooth shift, all regular checks completed..."
                height={250}
              />
            </div>

            <div className="space-y-2">
              <label className="block text-sm font-semibold text-orange-500">Pending Issues</label>
              <p className="text-xs text-muted-foreground mb-1">What is left for the next shift to handle?</p>
              <RichTextEditor
                value={pendingIssues}
                onChange={setPendingIssues}
                placeholder="E.g., Ticket #1234 is still open, waiting for customer reply."
                height={250}
              />
            </div>
          </div>

          <div className="p-6 border-t border-border bg-muted/30 flex justify-end gap-3">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-border text-foreground rounded-lg hover:bg-accent transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={mutation.isPending}
              className="px-6 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 disabled:opacity-50 transition-colors shadow-sm font-medium"
            >
              {mutation.isPending ? 'Submitting...' : 'Submit Handover'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
