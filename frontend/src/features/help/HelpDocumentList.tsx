import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { FileText, Plus, ShieldAlert } from 'lucide-react';
import { helpDocumentService } from '../../services/api/helpDocumentService';
import { useAuthStore } from '../../store/authStore';
import { format } from 'date-fns';
import { HelpPermissionsModal } from './HelpPermissionsModal';
import { DraggableGrid } from '../../components/ui/DraggableGrid';
import type { GridLayout } from '../../components/ui/DraggableGrid';
import { Search } from 'lucide-react';
import api from '@/lib/api';

export function HelpDocumentList() {
  const navigate = useNavigate();
  const { user, updateUserPreferences } = useAuthStore();
  const [isPermissionsModalOpen, setIsPermissionsModalOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [layout, setLayout] = useState<GridLayout>({ folders: {}, order: [] });

  useEffect(() => {
    if (user?.ui_preferences?.info_bank_layout) {
      setLayout(user.ui_preferences.info_bank_layout);
    }
  }, [user]);

  const handleLayoutChange = async (newLayout: GridLayout) => {
    setLayout(newLayout);
    const newPrefs = { info_bank_layout: newLayout };
    updateUserPreferences(newPrefs);
    if (user?.id) {
      try {
        await api.put(`/employees/${user.id}/preferences`, { ui_preferences: { ...user?.ui_preferences, ...newPrefs } });
      } catch (e) {
        console.error("Failed to save layout", e);
      }
    }
  };
  
  const { data: documents = [], isLoading } = useQuery({
    queryKey: ['help-docs'],
    queryFn: helpDocumentService.getVisibleDocuments,
  });

  const canCreate = user?.role === 'manager' || user?.role === 'team_leader' || user?.role === 'admin' || user?.can_manage_help_docs;
  const canManagePermissions = user?.role === 'manager' || user?.role === 'admin';

  if (isLoading) {
    return <div className="p-8 text-center text-gray-500">Loading...</div>;
  }

  const filteredDocuments = documents.filter(doc => 
    doc.title.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Info Bank</h1>
          <p className="text-gray-500 text-sm mt-1">Quick access to guides and instructions for your department</p>
        </div>
        
        <div className="flex items-center gap-2">
          {canManagePermissions && (
            <button
              onClick={() => setIsPermissionsModalOpen(true)}
              className="flex items-center gap-2 bg-white text-gray-700 border border-gray-300 px-4 py-2 rounded-lg hover:bg-gray-50 transition-colors"
            >
              <ShieldAlert className="w-4 h-4" />
              <span>Manage Permissions</span>
            </button>
          )}
          {canCreate && (
            <button
              onClick={() => navigate('/help/new')}
              className="flex items-center gap-2 bg-indigo-600 text-white px-4 py-2 rounded-lg hover:bg-indigo-700 transition-colors"
            >
              <Plus className="w-4 h-4" />
              <span>New Document</span>
            </button>
          )}
        </div>
      </div>
      
      <HelpPermissionsModal 
        isOpen={isPermissionsModalOpen} 
        onClose={() => setIsPermissionsModalOpen(false)} 
      />

      <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-4">
        <div className="relative max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            type="text"
            placeholder="Search documents..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg bg-gray-50 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 transition-all"
          />
        </div>
      </div>

      {documents.length === 0 ? (
        <div className="bg-white rounded-xl border border-gray-200 p-12 text-center">
          <div className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <FileText className="w-8 h-8 text-gray-400" />
          </div>
          <h3 className="text-lg font-medium text-gray-900 mb-1">No Documents</h3>
          <p className="text-gray-500 max-w-sm mx-auto">
            No help documents have been added for your department yet, or you don't have access to them.
          </p>
        </div>
      ) : (
        <DraggableGrid
          items={filteredDocuments}
          layout={layout}
          onLayoutChange={handleLayoutChange}
          isSearchActive={searchTerm.length > 0}
          onItemClick={(doc) => navigate(`/help/${doc.id}`)}
          renderItem={(doc) => (
            <div 
              className="bg-white rounded-xl border border-gray-200 p-6 hover:shadow-md hover:border-indigo-200 transition-all cursor-pointer group h-full flex flex-col"
            >
              <div className="flex items-start justify-between mb-4">
                <div className="p-3 bg-indigo-50 text-indigo-600 rounded-lg group-hover:bg-indigo-600 group-hover:text-white transition-colors">
                  <FileText className="w-6 h-6" />
                </div>
                {doc.access_level === 'write' && (
                  <span className="text-xs font-medium bg-amber-100 text-amber-700 px-2 py-1 rounded-full flex items-center gap-1">
                    <ShieldAlert className="w-3 h-3" />
                    Editor
                  </span>
                )}
              </div>
              
              <h3 className="text-lg font-bold text-gray-900 mb-2 line-clamp-2 flex-grow">{doc.title}</h3>
              
              <div className="text-xs text-gray-500 mt-4 flex items-center justify-between border-t border-gray-100 pt-4">
                <span>Created at: {format(new Date(doc.created_at), 'yyyy-MM-dd')}</span>
              </div>
            </div>
          )}
        />
      )}
    </div>
  );
}
