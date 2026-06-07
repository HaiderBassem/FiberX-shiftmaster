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
    <div className="max-w-5xl mx-auto space-y-6 pb-12">
      {/* Header Section */}
      <div className="bg-card rounded-2xl shadow-sm border border-border p-6 flex flex-col sm:flex-row sm:items-center justify-between gap-6">
        <div className="flex items-start sm:items-center gap-4">
          <button
            onClick={() => navigate('/help')}
            className="p-2.5 bg-muted/50 hover:bg-muted rounded-full transition-colors text-muted-foreground hover:text-foreground shrink-0 mt-1 sm:mt-0"
            title="Back to Info Bank"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <div>
            <h1 className="text-2xl sm:text-3xl font-bold text-foreground tracking-tight leading-tight">
              {document.title}
            </h1>
            <p className="text-sm text-muted-foreground mt-2 flex items-center gap-2">
              <span className="inline-block w-2 h-2 rounded-full bg-primary/60"></span>
              Created on {format(new Date(document.created_at), 'MMMM d, yyyy')}
            </p>
          </div>
        </div>
        
        <div className="flex flex-wrap items-center gap-3">
          {canManageAccess && (
            <button
              onClick={() => setShowAccessManager(!showAccessManager)}
              className="flex items-center gap-2 bg-secondary/50 text-secondary-foreground border border-border px-4 py-2.5 rounded-xl hover:bg-secondary transition-colors font-medium text-sm"
            >
              <Users className="w-4 h-4" />
              <span>Manage Access</span>
            </button>
          )}
          {canEdit && (
            <button
              onClick={() => navigate(`/help/${id}/edit`)}
              className="flex items-center gap-2 bg-primary text-primary-foreground px-5 py-2.5 rounded-xl hover:bg-primary/90 transition-all font-medium shadow-sm hover:shadow-md text-sm"
            >
              <Edit className="w-4 h-4" />
              <span>Edit Document</span>
            </button>
          )}
        </div>
      </div>

      {showAccessManager && canManageAccess && (
        <div className="bg-card rounded-2xl shadow-sm border border-border p-6 animate-in slide-in-from-top-4 fade-in duration-300">
          <HelpDocumentAccessManager documentId={id!} />
        </div>
      )}

      {/* Document Content */}
      <div className="bg-card rounded-2xl shadow-sm border border-border overflow-hidden">
        {document.access_level === 'write' && (
          <div className="bg-amber-500/10 border-b border-amber-500/20 p-4 px-6 flex items-center gap-3 text-amber-600 text-sm font-medium">
            <ShieldAlert className="w-5 h-5" />
            <span>You have permission to edit this document. Your changes will be visible to all permitted users.</span>
          </div>
        )}
        
        <div 
          className="p-8 sm:p-10 lg:p-12 prose prose-slate dark:prose-invert max-w-none prose-headings:font-bold prose-h1:text-3xl prose-a:text-primary prose-img:rounded-xl prose-img:shadow-sm jodit-content"
          dir="auto"
          dangerouslySetInnerHTML={{ __html: document.content }}
        />
      </div>
    </div>
  );
}
