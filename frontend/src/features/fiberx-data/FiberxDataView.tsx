import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams, useNavigate } from 'react-router-dom';
import { Edit2, ArrowLeft, Trash2, Calendar, User, Database, Share2 } from 'lucide-react';
import { fiberxDataService } from '../../services/fiberxDataService';
import { format } from 'date-fns';

export function FiberxDataView() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: doc, isLoading } = useQuery({
    queryKey: ['fiberx-data', id],
    queryFn: () => fiberxDataService.getById(id!),
    enabled: !!id,
  });

  const deleteMutation = useMutation({
    mutationFn: () => fiberxDataService.delete(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fiberx-data'] });
      navigate('/fiberx-data');
    },
  });

  const handleDelete = () => {
    if (window.confirm('Are you sure you want to delete this document?')) {
      deleteMutation.mutate();
    }
  };

  if (isLoading) return <div className="p-8 text-center text-muted-foreground">Loading document...</div>;
  if (!doc) return <div className="p-8 text-center text-destructive">Document not found or you don't have access.</div>;

  const canEdit = doc.access_level === 'write';

  return (
    <div className="w-full px-4 md:px-8 mx-auto space-y-6 pb-12">
      <div className="flex items-center justify-between">
        <button
          onClick={() => navigate('/fiberx-data')}
          className="flex items-center gap-2 text-muted-foreground hover:text-foreground transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          <span>Back to Data</span>
        </button>
        
        {canEdit && (
          <div className="flex items-center gap-2">
            <button
              onClick={() => navigate(`/fiberx-data/${id}/edit`)}
              className="flex items-center gap-2 bg-primary text-primary-foreground px-4 py-2 rounded-lg hover:bg-primary/90 transition-colors"
            >
              <Edit2 className="w-4 h-4" />
              <span>Edit Document</span>
            </button>
            <button
              onClick={handleDelete}
              disabled={deleteMutation.isPending}
              className="flex items-center gap-2 bg-destructive/10 text-destructive px-4 py-2 rounded-lg hover:bg-destructive hover:text-destructive-foreground transition-colors"
            >
              <Trash2 className="w-4 h-4" />
              <span>Delete</span>
            </button>
          </div>
        )}
      </div>

      <article className="bg-card rounded-2xl shadow-sm border border-border overflow-hidden">
        <div className="border-b border-border p-8 bg-muted/30">
          <div className="flex items-center gap-3 mb-4">
            <span className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-sm font-medium ${
              doc.is_shared ? 'bg-purple-500/10 text-purple-600' : 'bg-indigo-500/10 text-indigo-600'
            }`}>
              {doc.is_shared ? <Share2 className="w-4 h-4" /> : <Database className="w-4 h-4" />}
              {doc.department_name}
            </span>
          </div>
          <h1 className="text-3xl font-bold text-foreground mb-4">{doc.title}</h1>
          <div className="flex flex-wrap items-center gap-6 text-sm text-muted-foreground">
            <div className="flex items-center gap-2">
              <User className="w-4 h-4" />
              <span>{doc.creator_name}</span>
            </div>
            <div className="flex items-center gap-2">
              <Calendar className="w-4 h-4" />
              <span>Created {format(new Date(doc.created_at), 'MMMM d, yyyy')}</span>
            </div>
            {doc.updated_at !== doc.created_at && (
              <div className="flex items-center gap-2">
                <Edit2 className="w-4 h-4" />
                <span>Updated {format(new Date(doc.updated_at), 'MMMM d, yyyy')}</span>
              </div>
            )}
          </div>
        </div>
        
        <div 
          className="p-8 prose prose-slate dark:prose-invert max-w-none prose-headings:font-semibold prose-a:text-primary jodit-content"
          dir="auto"
          dangerouslySetInnerHTML={{ __html: doc.content }}
        />
      </article>
    </div>
  );
}
