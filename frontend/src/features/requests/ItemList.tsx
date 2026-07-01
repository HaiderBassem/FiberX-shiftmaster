import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Plus, Package, Clock, Settings, PackageOpen } from 'lucide-react';
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

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold flex items-center gap-2">
            <Package className="w-5 h-5 text-primary" />
            Company Items
          </h3>
          <p className="text-sm text-muted-foreground">Request items like laptops, stationery, etc.</p>
        </div>
        <div className="flex gap-2">
          {isSupervisor && (
            <Button variant="outline" onClick={() => setIsCategoryManagerOpen(true)}>
              <Settings className="w-4 h-4 mr-2" />
              Manage Categories
            </Button>
          )}
          <Button onClick={() => setIsRequestModalOpen(true)}>
            <Plus className="w-4 h-4 mr-2" />
            Request Item
          </Button>
        </div>
      </div>

      <div className="grid gap-4">
        {isLoading ? (
          <div className="h-32 flex items-center justify-center">
            <div className="w-6 h-6 border-2 border-primary border-t-transparent rounded-full animate-spin" />
          </div>
        ) : requests && requests.length > 0 ? (
          requests.map((req) => (
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
                <div>
                  <span className={`px-2.5 py-1 text-xs font-semibold rounded-full capitalize border ${
                    req.status === 'pending' 
                      ? 'bg-amber-500/10 text-amber-600 border-amber-500/20'
                      : req.status === 'processed'
                      ? 'bg-emerald-500/10 text-emerald-600 border-emerald-500/20'
                      : 'bg-muted text-muted-foreground border-border'
                  }`}>
                    {req.status}
                  </span>
                </div>
              </CardContent>
            </Card>
          ))
        ) : (
          <div className="text-center py-12 border-2 border-dashed border-border rounded-xl bg-muted/20">
            <PackageOpen className="w-12 h-12 text-muted-foreground/30 mx-auto mb-3" />
            <p className="text-muted-foreground">You haven't requested any items yet.</p>
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
