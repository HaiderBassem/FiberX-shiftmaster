import React, { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import ReactQuill from 'react-quill';
import 'react-quill/dist/quill.snow.css';
import { ArrowRight, Save, Trash2 } from 'lucide-react';
import { helpDocumentService } from '../../services/api/helpDocumentService';

const modules = {
  toolbar: [
    [{ 'header': [1, 2, 3, false] }],
    ['bold', 'italic', 'underline', 'strike'],
    [{ 'color': [] }, { 'background': [] }],
    [{ 'list': 'ordered'}, { 'list': 'bullet' }],
    [{ 'align': [] }],
    [{ 'direction': 'rtl' }],
    ['clean']
  ],
};

const formats = [
  'header',
  'bold', 'italic', 'underline', 'strike',
  'color', 'background',
  'list', 'bullet',
  'align', 'direction'
];

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
      alert('الرجاء إدخال عنوان المستند');
      return;
    }
    saveMutation.mutate();
  };

  const handleDelete = () => {
    if (window.confirm('هل أنت متأكد من حذف هذا المستند؟')) {
      deleteMutation.mutate();
    }
  };

  if (isLoading) return <div className="p-8 text-center text-gray-500">جاري التحميل...</div>;

  // Enforce write access check if editing
  if (isEditing && document && document.access_level !== 'write') {
    return (
      <div className="p-8 text-center text-red-500">
        عذراً، ليس لديك صلاحية تعديل هذا المستند.
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <button
            onClick={() => navigate('/help')}
            className="p-2 hover:bg-gray-100 rounded-full transition-colors text-gray-500"
          >
            <ArrowRight className="w-5 h-5" />
          </button>
          <h1 className="text-2xl font-bold text-gray-900">
            {isEditing ? 'تعديل المستند' : 'إنشاء مستند جديد'}
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
              <span>حذف</span>
            </button>
          )}
          <button
            onClick={handleSave}
            disabled={saveMutation.isPending || !title.trim()}
            className="flex items-center gap-2 bg-indigo-600 text-white px-6 py-2 rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
          >
            <Save className="w-4 h-4" />
            <span>{saveMutation.isPending ? 'جاري الحفظ...' : 'حفظ المستند'}</span>
          </button>
        </div>
      </div>

      <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
        <div className="p-6 border-b border-gray-100">
          <input
            type="text"
            placeholder="عنوان المستند..."
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="w-full text-2xl font-bold bg-transparent border-none focus:ring-0 p-0 text-gray-900 placeholder:text-gray-300 outline-none"
            dir="auto"
          />
        </div>
        
        <div className="bg-white min-h-[500px]" dir="auto">
          <ReactQuill
            theme="snow"
            value={content}
            onChange={setContent}
            modules={modules}
            formats={formats}
            className="h-full min-h-[450px]"
            placeholder="ابدأ الكتابة هنا..."
          />
        </div>
      </div>
      
      {/* Add custom CSS to fix ReactQuill RTL alignment if needed */}
      <style dangerouslySetInnerHTML={{__html: `
        .ql-editor {
          min-height: 450px;
          font-size: 16px;
          line-height: 1.6;
        }
        .ql-editor[dir="rtl"] {
          text-align: right;
        }
      `}} />
    </div>
  );
}
