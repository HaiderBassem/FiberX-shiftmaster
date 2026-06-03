import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, Edit, ShieldAlert, Users } from 'lucide-react';
import { helpDocumentService } from '../../services/api/helpDocumentService';
import { useAuthStore } from '../../store/authStore';
import { format } from 'date-fns';
import { HelpDocumentAccessManager } from './HelpDocumentAccessManager';

export function HelpDocumentView() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { user } = useAuthStore();
  const [showAccessManager, setShowAccessManager] = useState(false);

  const { data: document, isLoading } = useQuery({
    queryKey: ['help-docs', id],
    queryFn: () => helpDocumentService.getDocument(id!),
  });

  if (isLoading) return <div className="p-8 text-center text-gray-500">Loading...</div>;
  if (!document) return <div className="p-8 text-center text-red-500">Document not found or you don't have permission to view it.</div>;

  const canEdit = document.access_level === 'write';
  const canManageAccess = user?.role === 'manager' || user?.role === 'admin';

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          <button
            onClick={() => navigate('/help')}
            className="p-2 hover:bg-gray-100 rounded-full transition-colors text-gray-500"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <div>
            <h1 className="text-2xl font-bold text-gray-900">{document.title}</h1>
            <p className="text-sm text-gray-500 mt-1">
              Created on {format(new Date(document.created_at), 'yyyy-MM-dd')}
            </p>
          </div>
        </div>
        
        <div className="flex items-center gap-3">
          {canManageAccess && (
            <button
              onClick={() => setShowAccessManager(!showAccessManager)}
              className="flex items-center gap-2 bg-white border border-gray-200 text-gray-700 px-4 py-2 rounded-lg hover:bg-gray-50 transition-colors"
            >
              <Users className="w-4 h-4" />
              <span>Manage Access</span>
            </button>
          )}
          {canEdit && (
            <button
              onClick={() => navigate(`/help/${id}/edit`)}
              className="flex items-center gap-2 bg-indigo-600 text-white px-4 py-2 rounded-lg hover:bg-indigo-700 transition-colors"
            >
              <Edit className="w-4 h-4" />
              <span>Edit</span>
            </button>
          )}
        </div>
      </div>

      {showAccessManager && canManageAccess && (
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
          <HelpDocumentAccessManager documentId={id!} />
        </div>
      )}

      <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
        {document.access_level === 'write' && (
          <div className="bg-amber-50 border-b border-amber-100 p-3 px-6 flex items-center gap-2 text-amber-800 text-sm">
            <ShieldAlert className="w-4 h-4" />
            <span>You have permission to edit this document.</span>
          </div>
        )}
        
        <div 
          className="p-8 max-w-none text-gray-800 text-lg leading-relaxed whitespace-pre-wrap break-words"
          style={{ fontFamily: "'Inter', 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif" }}
          dir="auto"
        >
          {document.content}
        </div>
      </div>
    </div>
  );
}
