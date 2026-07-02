import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { X } from 'lucide-react';
import { RichTextEditor } from '@/components/RichTextEditor';
import { useTranslation } from 'react-i18next';

export default function CreateHandoverModal({
  isOpen,
  onClose,
}: {
  isOpen: boolean;
  onClose: () => void;
}) {
  const { t } = useTranslation();
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
      setError(err.response?.data?.error || t('handovers.failed_create_handover'));
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!shiftSummary.trim()) {
      setError(t('handovers.shift_summary_required'));
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
          <h2 className="text-xl font-semibold text-foreground">{t('handovers.create_shift_handover')}</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="p-6 space-y-4">
            {error && <div className="p-3 bg-red-500/10 text-red-500 rounded-md text-sm">{error}</div>}

            <div className="space-y-2">
              <label className="block text-sm font-semibold text-foreground">{t('handovers.shift_summary')} <span className="text-destructive">*</span></label>
              <p className="text-xs text-muted-foreground mb-1">{t('handovers.what_happened')}</p>
              <RichTextEditor
                value={shiftSummary}
                onChange={setShiftSummary}
                placeholder={t('handovers.eg_smooth_shift')}
                height={250}
              />
            </div>

            <div className="space-y-2">
              <label className="block text-sm font-semibold text-orange-500">{t('handovers.pending_issues')}</label>
              <p className="text-xs text-muted-foreground mb-1">{t('handovers.what_is_left')}</p>
              <RichTextEditor
                value={pendingIssues}
                onChange={setPendingIssues}
                placeholder={t('handovers.eg_ticket_open')}
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
              {t('common.cancel')}
            </button>
            <button
              type="submit"
              disabled={mutation.isPending}
              className="px-6 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 disabled:opacity-50 transition-colors shadow-sm font-medium"
            >
              {mutation.isPending ? t('common.submitting') : t('handovers.submit_handover')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
