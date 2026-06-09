import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { Database, Plus, ShieldAlert, Search, Folder, Share2 } from 'lucide-react';
import { fiberxDataService } from '../../services/fiberxDataService';
import { useAuthStore } from '../../store/authStore';
import { format } from 'date-fns';
import { FiberxDataPermissionsModal } from './FiberxDataPermissionsModal';
import { FiberxDataGlobalPermissionsModal } from './FiberxDataGlobalPermissionsModal';
import type { FiberxData } from '@/services/fiberxDataService';
import { Button } from '@/components/ui/button';

export function FiberxDataHub() {
  const navigate = useNavigate();
  const { user } = useAuthStore();
  const [selectedDocId, setSelectedDocId] = useState<string | null>(null);
  const [isPermissionsModalOpen, setIsPermissionsModalOpen] = useState(false);
  const [isGlobalPermissionsOpen, setIsGlobalPermissionsOpen] = useState(false);
  const [selectedDocIdForPerms, setSelectedDocIdForPerms] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [activeTab, setActiveTab] = useState<'my-dept' | 'shared'>('my-dept');

  const { data = [], isLoading } = useQuery({
    queryKey: ['fiberx-data'],
    queryFn: fiberxDataService.getVisible,
  });

  const documents = Array.isArray(data) ? data : [];

  const canCreate = user?.role === 'manager' || user?.role === 'admin' || user?.role === 'team_leader' || (user as any)?.can_manage_fiberx_data;
  const canManagePermissions = user?.role === 'manager' || user?.role === 'admin' || user?.role === 'team_leader';

  if (isLoading) {
    return <div className="p-8 text-center text-muted-foreground">Loading...</div>;
  }

  const filteredDocuments = documents.filter(doc => 
    doc.title.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const myDeptDocs = filteredDocuments.filter(d => !d.is_shared);
  const sharedDocs = filteredDocuments.filter(d => d.is_shared);

  const docsToDisplay = activeTab === 'my-dept' ? myDeptDocs : sharedDocs;

  const openPermissions = (e: React.MouseEvent, docId: string) => {
    e.stopPropagation();
    setSelectedDocIdForPerms(docId);
    setIsPermissionsModalOpen(true);
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-2xl font-bold text-foreground">FiberX Data</h1>
          <p className="text-muted-foreground text-sm mt-1">Manage and access centralized data and shared documents</p>
        </div>
        
        <div className="flex items-center gap-2">
          {canManagePermissions && (
            <Button
              variant="outline"
              onClick={() => setIsGlobalPermissionsOpen(true)}
              className="gap-2"
            >
              <ShieldAlert className="w-4 h-4" />
              <span>Global Permissions</span>
            </Button>
          )}
          {canCreate && (
            <Button
              onClick={() => navigate('/fiberx-data/new')}
              className="gap-2 bg-indigo-600 hover:bg-indigo-700 text-white"
            >
              <Plus className="w-4 h-4" />
              <span>New Document</span>
            </Button>
          )}
        </div>
      </div>
      
      {isPermissionsModalOpen && selectedDocIdForPerms && (
        <FiberxDataPermissionsModal 
          isOpen={isPermissionsModalOpen} 
          onClose={() => {
            setIsPermissionsModalOpen(false);
            setSelectedDocIdForPerms(null);
          }} 
          documentId={selectedDocIdForPerms}
        />
      )}

      <FiberxDataGlobalPermissionsModal
        isOpen={isGlobalPermissionsOpen}
        onClose={() => setIsGlobalPermissionsOpen(false)}
      />

      <div className="bg-card rounded-xl shadow-sm border border-border p-4 flex flex-col sm:flex-row gap-4 justify-between items-center">
        <div className="flex p-1 bg-muted rounded-lg w-full sm:w-auto">
          <button
            onClick={() => setActiveTab('my-dept')}
            className={`flex-1 sm:flex-none px-4 py-1.5 text-sm font-medium rounded-md transition-colors ${
              activeTab === 'my-dept' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            My Department
          </button>
          <button
            onClick={() => setActiveTab('shared')}
            className={`flex-1 sm:flex-none px-4 py-1.5 text-sm font-medium rounded-md transition-colors ${
              activeTab === 'shared' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            Shared with Us
          </button>
        </div>

        <div className="relative w-full sm:max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search data..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-input rounded-lg bg-background focus:ring-2 focus:ring-primary focus:border-primary transition-all text-sm"
          />
        </div>
      </div>

      {docsToDisplay.length === 0 ? (
        <div className="bg-card rounded-xl border border-dashed border-border p-12 text-center">
          <div className="w-16 h-16 bg-muted rounded-full flex items-center justify-center mx-auto mb-4">
            <Database className="w-8 h-8 text-muted-foreground" />
          </div>
          <h3 className="text-lg font-medium text-foreground mb-1">No Data Found</h3>
          <p className="text-muted-foreground max-w-sm mx-auto">
            {activeTab === 'my-dept' 
              ? "Your department hasn't created any data documents yet."
              : "No data documents have been shared with your department yet."}
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {docsToDisplay.map((doc) => (
            <div 
              key={doc.id}
              onClick={() => navigate(`/fiberx-data/${doc.id}`)}
              className="bg-card rounded-xl border border-border p-6 hover:shadow-md hover:border-primary/50 transition-all cursor-pointer group flex flex-col h-full relative"
            >
              <div className="flex items-start justify-between mb-4">
                <div className={`p-3 rounded-lg transition-colors ${
                  doc.is_shared 
                    ? 'bg-purple-500/10 text-purple-600 group-hover:bg-purple-600 group-hover:text-white'
                    : 'bg-indigo-500/10 text-indigo-600 group-hover:bg-indigo-600 group-hover:text-white'
                }`}>
                  {doc.is_shared ? <Share2 className="w-6 h-6" /> : <Database className="w-6 h-6" />}
                </div>
                <div className="flex gap-2">
                  {doc.access_level === 'write' && (
                    <span className="text-xs font-medium bg-amber-500/10 text-amber-600 px-2 py-1 rounded-full flex items-center gap-1">
                      Editor
                    </span>
                  )}
                  {canManagePermissions && !doc.is_shared && (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 px-2 text-xs text-muted-foreground hover:text-foreground"
                      onClick={(e) => openPermissions(e, doc.id)}
                    >
                      <ShieldAlert className="w-3.5 h-3.5 mr-1" />
                      Manage
                    </Button>
                  )}
                </div>
              </div>
              
              <h3 className="text-lg font-bold text-foreground mb-2 line-clamp-2">{doc.title}</h3>
              
              <div className="text-xs text-muted-foreground mt-auto pt-4 flex flex-col gap-1 border-t border-border/50">
                <div className="flex items-center gap-1.5">
                  <Folder className="w-3.5 h-3.5" />
                  <span>{doc.department_name}</span>
                </div>
                <div className="flex items-center justify-between mt-1">
                  <span>By {doc.creator_name}</span>
                  <span>{format(new Date(doc.created_at), 'MMM d, yyyy')}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
