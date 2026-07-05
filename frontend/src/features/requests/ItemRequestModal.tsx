import { useState } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { X, Send } from 'lucide-react';
import { Button } from '../../components/ui/button';
import api from '../../lib/api';
import { useTranslation } from 'react-i18next';

interface Category {
  id: string;
  name: string;
}

interface ItemRequestModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export const ItemRequestModal = ({ isOpen, onClose, onSuccess }: ItemRequestModalProps) => {
  const { t } = useTranslation();
  const [categoryId, setCategoryId] = useState('');
  const [description, setDescription] = useState('');

  const { data: categories, isLoading } = useQuery({
    queryKey: ['item-categories'],
    queryFn: async () => {
      const { data } = await api.get('/item-requests/categories');
      return data.data as Category[];
    },
    enabled: isOpen,
  });

  const submitMutation = useMutation({
    mutationFn: async () => {
      await api.post('/item-requests', {
        category_id: categoryId,
        description,
      });
    },
    onSuccess: () => {
      onSuccess();
      onClose();
      setCategoryId('');
      setDescription('');
    },
  });

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-background/80 backdrop-blur-sm">
      <div className="bg-card w-full max-w-md rounded-2xl shadow-xl border border-border overflow-hidden animate-in zoom-in-95 duration-200">
        <div className="flex items-center justify-between p-4 border-b border-border">
          <h2 className="text-lg font-semibold text-foreground">{t('items.request_company_item')}</h2>
          <button onClick={onClose} className="p-2 hover:bg-muted rounded-full transition-colors text-muted-foreground">
            <X className="w-5 h-5" />
          </button>
        </div>
        
        <form onSubmit={(e) => {
          e.preventDefault();
          if (categoryId && description.trim()) {
            submitMutation.mutate();
          }
        }} className="p-4 space-y-4">
          
          <div>
            <label className="block text-sm font-medium text-foreground mb-1.5">{t('items.item_category')}</label>
            <select
              required
              value={categoryId}
              onChange={(e) => setCategoryId(e.target.value)}
              className="w-full px-3 py-2 bg-background border border-input rounded-xl focus:ring-2 focus:ring-primary/20 focus:border-primary transition-colors text-foreground appearance-none"
            >
              <option value="">{t('items.select_category')}</option>
              {categories?.map((cat) => (
                <option key={cat.id} value={cat.id}>{cat.name}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-foreground mb-1.5">{t('items.description_reason')}</label>
            <textarea
              required
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full px-3 py-2 bg-background border border-input rounded-xl focus:ring-2 focus:ring-primary/20 focus:border-primary transition-colors text-foreground resize-none"
              rows={4}
              placeholder={t('items.item_placeholder')}
            />
          </div>

          <div className="pt-2 flex justify-end gap-3">
            <Button type="button" variant="outline" onClick={onClose}>
              {t('common.cancel')}
            </Button>
            <Button 
              type="submit" 
              disabled={submitMutation.isPending || isLoading || !categoryId || !description.trim()}
            >
              <Send className="w-4 h-4 mr-2" />
              {submitMutation.isPending ? t('items.submitting') : t('items.submit_request')}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
};
