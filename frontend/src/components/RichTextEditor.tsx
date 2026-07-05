import React, { useRef, useMemo } from 'react';
import JoditEditor from 'jodit-react';
import api from '../lib/api';
import { useAuthStore } from '../store/authStore';

interface RichTextEditorProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  height?: number;
}

export const RichTextEditor: React.FC<RichTextEditorProps> = ({ 
  value, 
  onChange, 
  placeholder,
  height = 400
}) => {
  const editor = useRef(null);
  const token = useAuthStore((state) => state.token);

  // Configuration for Jodit Editor
  const config = useMemo(() => ({
    readonly: false,
    placeholder: placeholder || 'Start typing...',
    height: height,
    width: '100%',
    askBeforePasteHTML: false,
    defaultActionOnPaste: 'insert_as_html' as const,
    askBeforePasteFromWord: false,
    direction: '' as const, // allows both RTL and LTR
    toolbarAdaptive: true,
    buttons: [
      'bold', 'italic', 'underline', '|',
      'ul', 'ol', '|',
      'font', 'fontsize', 'brush', '|',
      'image', 'table', 'link', '|',
      'align', 'undo', 'redo', '|',
      'copyformat', 'fullsize'
    ],
    uploader: {
      insertImageAsBase64URI: false,
      url: `${api.defaults.baseURL || '/api'}/upload/image`, // Correctly resolve API base URL
      format: 'json',
      headers: {
        Authorization: `Bearer ${token}`, // Pass the JWT token for auth
      },
      isSuccess: (resp: any) => {
        return resp && resp.success;
      },
      process: (resp: any) => {
        // Jodit expects this format by default if we don't map it properly, 
        // but we return {"success": true, "data": {"url": "..."}}
        if (resp && resp.success && resp.data && resp.data.url) {
          return {
            files: [resp.data.url],
            path: resp.data.url,
            baseurl: '',
            error: 0,
            msg: ''
          };
        }
        return resp;
      },
      defaultHandlerSuccess: function (data: any) {
        // 'this' is bound to Jodit's uploader instance inside defaultHandlerSuccess
        // We need to access the Jodit instance.
        const j = (this as any).jodit;
        if (data && data.files && data.files.length) {
          data.files.forEach((url: string) => {
            j.selection.insertImage(url, null, j.options.imageDefaultWidth);
          });
        }
      },
      error: (e: any) => {
        console.error('Image upload failed:', e);
      }
    },
    controls: {
      font: {
        list: {
          'Inter, sans-serif': 'Inter',
          'Roboto, sans-serif': 'Roboto',
          'Outfit, sans-serif': 'Outfit',
        }
      }
    },
    style: {
      background: 'transparent',
      color: 'inherit'
    },
    theme: 'default',
    enableDragAndDropFileToEditor: true,
  }), [placeholder, height, token]);

  return (
    <div className="rich-text-editor-container w-full h-full" style={{ borderRadius: '0.5rem', overflow: 'hidden' }}>
      <JoditEditor
        ref={editor}
        value={value}
        config={config}
        tabIndex={1} // tabIndex of textarea
        onBlur={newContent => onChange(newContent)} // preferred to use only this option to update the content for performance reasons
        onChange={() => {}} // empty to prevent unnecessary re-renders during typing
      />
    </div>
  );
};
