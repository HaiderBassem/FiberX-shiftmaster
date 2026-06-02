import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { FileText, Plus, ShieldAlert } from 'lucide-react';
import { helpDocumentService } from '../../services/api/helpDocumentService';
import { useAuthStore } from '../../store/authStore';
import { format } from 'date-fns';

export function HelpDocumentList() {
  const navigate = useNavigate();
  const { user } = useAuthStore();
  
  const { data: documents = [], isLoading } = useQuery({
    queryKey: ['help-docs'],
    queryFn: helpDocumentService.getVisibleDocuments,
  });

  const canCreate = user?.role === 'manager' || user?.role === 'team_leader' || user?.role === 'admin' || user?.can_manage_help_docs;

  if (isLoading) {
    return <div className="p-8 text-center text-gray-500">Loading...</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Help Documents</h1>
          <p className="text-gray-500 text-sm mt-1">Quick access to guides and instructions for your department</p>
        </div>
        
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
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {documents.map((doc) => (
            <div 
              key={doc.id}
              onClick={() => navigate(`/help/${doc.id}`)}
              className="bg-white rounded-xl border border-gray-200 p-6 hover:shadow-md hover:border-indigo-200 transition-all cursor-pointer group"
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
              
              <h3 className="text-lg font-bold text-gray-900 mb-2 line-clamp-1">{doc.title}</h3>
              
              <div className="text-xs text-gray-500 mt-4 flex items-center justify-between border-t border-gray-100 pt-4">
                <span>Created at: {format(new Date(doc.created_at), 'yyyy-MM-dd')}</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
