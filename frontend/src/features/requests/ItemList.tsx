import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Package, Clock, Settings, XCircle } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Card, CardContent } from '../../components/ui/card';
import { Button } from '../../components/ui/button';
import { useAuthStore } from '../../store/authStore';
import api from '../../lib/api';
import { ItemRequestModal } from './ItemRequestModal';
import { ItemCategoryManager } from './ItemCategoryManager';

interface ItemRequest {
  id: string;
  category_name: string;
  description: string;
  status: string;
  created_at: string;
}

export const ItemList = () => {
  const { user } = useAuthStore();
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const isSupervisor = ['admin', 'manager', 'team_leader'].includes(user?.role || '');

  const [isRequestModalOpen, setIsRequestModalOpen] = useState(false);
  const [isCategoryManagerOpen, setIsCategoryManagerOpen] = useState(false);

  const { data: requests, isLoading, refetch } = useQuery({
    queryKey: ['item-requests'],
    queryFn: async () => {
      const { data } = await api.get('/item-requests/me');
      return data.data as ItemRequest[];
    },
  });

  const cancelItemMutation = useMutation({
    mutationFn: async (id: string) => { await api.post(`/item-requests/${id}/cancel`); },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['item-requests'] }); },
    onError: (err: any) => alert(err?.response?.data?.error || err?.message || 'Failed to cancel request'),
  });

  const pendingRequests = requests?.filter(r => r.status === 'pending') || [];
  const historyRequests = requests?.filter(r => r.status !== 'pending') || [];

  const renderRequestCard = (req: ItemRequest) => (
    <Card key={req.id}>
      <CardContent className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h4 className="font-semibold text-foreground flex items-center gap-2">
            {req.category_name}
          </h4>
          <p className="text-sm text-muted-foreground mt-1 whitespace-pre-wrap">{req.description}</p>
          <div className="flex items-center gap-4 mt-3 text-xs text-muted-foreground">
            <span className="flex items-center gap-1">
              <Clock className="w-3.5 h-3.5" />
              {new Date(req.created_at).toLocaleDateString()}
            </span>
          </div>
        </div>
        <div className="flex items-center gap-3">
          {req.status === 'pending' && (
            <Button 
              variant="outline" 
              size="sm" 
              className="text-destructive border-destructive/30 hover:bg-destructive/10 h-8 text-xs px-2"
              onClick={() => {
                if (confirm(t('items.cancel_confirm'))) {
                  cancelItemMutation.mutate(req.id);
                }
              }}
              disabled={cancelItemMutation.isPending}
            >
              <XCircle className="w-3 h-3 mr-1" /> {t('common.cancel_request')}
            </Button>
          )}
          <span className={`px-2.5 py-1 text-xs font-semibold rounded-full capitalize border ${
            req.status === 'pending' 
              ? 'bg-amber-500/10 text-amber-600 border-amber-500/20'
              : req.status === 'processed'
              ? 'bg-emerald-500/10 text-emerald-600 border-emerald-500/20'
              : req.status === 'cancelled'
              ? 'bg-gray-500/10 text-gray-500 border-gray-500/20'
              : 'bg-muted text-muted-foreground border-border'
          }`}>
            {t(`common.${req.status}`)}
          </span>
        </div>
      </CardContent>
    </Card>
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold flex items-center gap-2">
            <Package className="w-5 h-5 text-primary" />
            {t('navigation.item_requests')}
          </h3>
        </div>
        <div className="flex gap-2">
          {isSupervisor && (
            <Button variant="outline" onClick={() => setIsCategoryManagerOpen(true)}>
              <Settings className="w-4 h-4 mr-2" />
              {t('items.manage_categories')}
            </Button>
          )}
          <Button onClick={() => setIsRequestModalOpen(true)}>
            <Plus className="w-4 h-4 mr-2" />
            {t('common.request')}
          </Button>
        </div>
      </div>

      <div className="grid gap-4">
        {isLoading ? (
          <div className="h-32 flex items-center justify-center">
            <div className="w-6 h-6 border-2 border-primary border-t-transparent rounded-full animate-spin" />
          </div>
        ) : (
          <div className="space-y-8">
            <div className="space-y-4">
              <h3 className="text-md font-semibold text-foreground tracking-tight">{t('common.pending')}</h3>
              {pendingRequests.length > 0 ? (
                pendingRequests.map(renderRequestCard)
              ) : (
                <div className="text-center py-6 border-2 border-dashed border-border rounded-xl bg-muted/20">
                  <p className="text-muted-foreground text-sm">{t('items.no_pending_requests')}</p>
                </div>
              )}
            </div>

            <div className="space-y-4">
              <h3 className="text-md font-semibold text-foreground tracking-tight">{t('common.history')}</h3>
              {historyRequests.length > 0 ? (
                historyRequests.map(renderRequestCard)
              ) : (
                <div className="text-center py-6 border-2 border-dashed border-border rounded-xl bg-muted/20">
                  <p className="text-muted-foreground text-sm">{t('items.no_history_available')}</p>
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      <ItemRequestModal 
        isOpen={isRequestModalOpen} 
        onClose={() => setIsRequestModalOpen(false)} 
        onSuccess={refetch}
      />
      
      {isSupervisor && (
        <ItemCategoryManager 
          isOpen={isCategoryManagerOpen} 
          onClose={() => setIsCategoryManagerOpen(false)} 
        />
      )}
    </div>
  );
};
