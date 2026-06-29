import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Save, Trash2 } from 'lucide-react';
import { helpDocumentService } from '../../services/api/helpDocumentService';
import { RichTextEditor } from '../../components/RichTextEditor';

export function HelpDocumentEditor() {
  const { id } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isEditing = !!id;

  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');

  const { data: document, isLoading } = useQuery({
    queryKey: ['help-docs', id],
    queryFn: () => helpDocumentService.getDocument(id!),
    enabled: isEditing,
  });

  useEffect(() => {
    if (document) {
      setTitle(document.title);
      setContent(document.content);
    }
  }, [document]);

  const saveMutation = useMutation({
    mutationFn: async () => {
      if (isEditing) {
        return helpDocumentService.updateDocument(id, { title, content });
      }
      return helpDocumentService.createDocument({ title, content });
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['help-docs'] });
      navigate(`/help/${data.id}`);
    }
  });

  const deleteMutation = useMutation({
    mutationFn: () => helpDocumentService.deleteDocument(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['help-docs'] });
      navigate('/help');
    }
  });

  const handleSave = () => {
    if (!title.trim()) {
      alert('Please enter a document title');
      return;
    }
    saveMutation.mutate();
  };

  const handleDelete = () => {
    if (window.confirm('Are you sure you want to delete this document?')) {
      deleteMutation.mutate();
    }
  };

  if (isLoading) return <div className="p-8 text-center text-gray-500">Loading...</div>;

  // Enforce write access check if editing
  if (isEditing && document && document.access_level !== 'write') {
    return (
      <div className="p-8 text-center text-red-500">
        Sorry, you do not have permission to edit this document.
      </div>
    );
  }

  return (
    <div className="w-full px-4 md:px-8 mx-auto space-y-6 pb-20">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <button
            onClick={() => navigate('/help')}
            className="p-2 hover:bg-gray-100 rounded-full transition-colors text-gray-500"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <h1 className="text-2xl font-bold text-gray-900">
            {isEditing ? 'Edit Document' : 'Create New Document'}
          </h1>
        </div>
        
        <div className="flex items-center gap-3">
          {isEditing && (
            <button
              onClick={handleDelete}
              disabled={deleteMutation.isPending}
              className="flex items-center gap-2 text-red-600 hover:bg-red-50 px-4 py-2 rounded-lg transition-colors"
            >
              <Trash2 className="w-4 h-4" />
              <span>Delete</span>
            </button>
          )}
          <button
            onClick={handleSave}
            disabled={saveMutation.isPending || !title.trim()}
            className="flex items-center gap-2 bg-indigo-600 text-white px-6 py-2 rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
          >
            <Save className="w-4 h-4" />
            <span>{saveMutation.isPending ? 'Saving...' : 'Save Document'}</span>
          </button>
        </div>
      </div>

      <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden flex flex-col h-[70vh]">
        <div className="p-6 border-b border-gray-100">
          <input
            type="text"
            placeholder="Document Title..."
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="w-full text-2xl font-bold bg-transparent border-none focus:ring-0 p-0 text-gray-900 placeholder:text-gray-300 outline-none font-sans"
            dir="auto"
          />
        </div>
        
        <div className="flex-1 p-0 overflow-y-auto">
          <RichTextEditor
            value={content}
            onChange={setContent}
            placeholder="Start typing your content here... (Supports images, tables, Arabic and English seamlessly)"
            height={window.innerHeight * 0.6}
          />
        </div>
      </div>
    </div>
  );
}
