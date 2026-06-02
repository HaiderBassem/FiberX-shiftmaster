import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { ArrowRight, Edit, ShieldAlert, Users } from 'lucide-react';
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

  if (isLoading) return <div className="p-8 text-center text-gray-500">جاري التحميل...</div>;
  if (!document) return <div className="p-8 text-center text-red-500">المستند غير موجود أو ليس لديك صلاحية للوصول إليه.</div>;

  const canEdit = document.access_level === 'write';
  const canManageAccess = user?.role === 'manager' || user?.role === 'team_leader' || user?.role === 'admin';

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          <button
            onClick={() => navigate('/help')}
            className="p-2 hover:bg-gray-100 rounded-full transition-colors text-gray-500"
          >
            <ArrowRight className="w-5 h-5" />
          </button>
          <div>
            <h1 className="text-2xl font-bold text-gray-900">{document.title}</h1>
            <p className="text-sm text-gray-500 mt-1">
              تم الإنشاء في {format(new Date(document.created_at), 'yyyy-MM-dd')}
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
              <span>إدارة الصلاحيات</span>
            </button>
          )}
          {canEdit && (
            <button
              onClick={() => navigate(`/help/${id}/edit`)}
              className="flex items-center gap-2 bg-indigo-600 text-white px-4 py-2 rounded-lg hover:bg-indigo-700 transition-colors"
            >
              <Edit className="w-4 h-4" />
              <span>تعديل</span>
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
            <span>لديك صلاحية لتعديل هذا المستند.</span>
          </div>
        )}
        
        <div 
          className="p-8 prose prose-indigo max-w-none prose-headings:font-bold"
          dir="auto"
          dangerouslySetInnerHTML={{ __html: document.content }}
        />
      </div>
      
      {/* Basic styles for quill content viewing */}
      <style dangerouslySetInnerHTML={{__html: `
        .prose img { max-width: 100%; height: auto; border-radius: 0.5rem; }
        .prose ul { list-style-type: disc; padding-inline-start: 1.5em; }
        .prose ol { list-style-type: decimal; padding-inline-start: 1.5em; }
        .prose a { color: #4f46e5; text-decoration: underline; }
        .prose blockquote { border-inline-start: 4px solid #e5e7eb; padding-inline-start: 1em; color: #6b7280; font-style: italic; }
      `}} />
    </div>
  );
}
