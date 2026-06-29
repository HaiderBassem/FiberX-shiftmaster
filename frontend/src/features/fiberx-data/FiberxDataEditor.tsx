import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Save, ArrowLeft } from 'lucide-react';
import { fiberxDataService } from '../../services/fiberxDataService';
import { Button } from '@/components/ui/button';
import { RichTextEditor } from '../../components/RichTextEditor';

export function FiberxDataEditor() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isEditing = Boolean(id);

  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');

  const { data: doc, isLoading } = useQuery({
    queryKey: ['fiberx-data', id],
    queryFn: () => fiberxDataService.getById(id!),
    enabled: isEditing,
  });

  useEffect(() => {
    if (doc) {
      setTitle(doc.title);
      setContent(doc.content);
    }
  }, [doc]);

  const saveMutation = useMutation({
    mutationFn: (data: { title: string; content: string }) => 
      isEditing ? fiberxDataService.update(id!, data) : fiberxDataService.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fiberx-data'] });
      navigate('/fiberx-data');
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim() || !content.trim()) return;
    saveMutation.mutate({ title, content });
  };

  if (isEditing && isLoading) {
    return <div className="p-8 text-center text-muted-foreground">Loading...</div>;
  }

  return (
    <div className="w-full px-4 md:px-8 mx-auto space-y-6 pb-20">
      <div className="flex items-center justify-between">
        <button
          onClick={() => navigate('/fiberx-data')}
          className="flex items-center gap-2 text-muted-foreground hover:text-foreground transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          <span>Back</span>
        </button>
      </div>

      <form onSubmit={handleSubmit} className="bg-card rounded-2xl shadow-sm border border-border p-8 space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">
            {isEditing ? 'Edit Document' : 'Create New Document'}
          </h1>
          <p className="text-muted-foreground text-sm mt-1">
            Write content using Markdown formatting
          </p>
        </div>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              Document Title
            </label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full px-4 py-2 bg-background border border-input rounded-lg focus:ring-2 focus:ring-primary focus:border-primary"
              placeholder="e.g., Q3 Analytics Report"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              Content
            </label>
            <div className="rounded-lg border border-input overflow-hidden">
              <RichTextEditor
                value={content}
                onChange={setContent}
                placeholder="Start typing your content here... (Supports images, tables, Arabic and English seamlessly)"
                height={500}
              />
            </div>
          </div>
        </div>

        <div className="flex justify-end gap-3 pt-4 border-t border-border">
          <Button
            type="button"
            variant="outline"
            onClick={() => navigate('/fiberx-data')}
          >
            Cancel
          </Button>
          <Button
            type="submit"
            disabled={saveMutation.isPending || !title.trim() || !content.trim()}
            className="gap-2"
          >
            <Save className="w-4 h-4" />
            <span>{saveMutation.isPending ? 'Saving...' : 'Save Document'}</span>
          </Button>
        </div>
      </form>
    </div>
  );
}
