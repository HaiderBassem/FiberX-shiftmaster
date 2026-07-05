import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { 
  Table as TableIcon, 
  Plus, 
  Search, 
  MoreVertical, 
  FileText,
  Lock,
  Globe,
  Shield
} from 'lucide-react';
import { infoTableService } from '../../services/api/infoTableService';
import type { InfoTable } from '../../types/infoTable';
import { useAuthStore } from '@/store/authStore';
import CreateTableModal from './CreateTableModal';
import { TablePermissionsModal } from './TablePermissionsModal';
import { DraggableGrid } from '../../components/ui/DraggableGrid';
import type { GridLayout } from '../../components/ui/DraggableGrid';
import api from '@/lib/api';

const InfoTableHub: React.FC = () => {
  const [tables, setTables] = useState<InfoTable[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isPermissionsModalOpen, setIsPermissionsModalOpen] = useState(false);
  const { user, updateUserPreferences } = useAuthStore();
  const navigate = useNavigate();
  const [layout, setLayout] = useState<GridLayout>({ folders: {}, order: [] });

  useEffect(() => {
    if (user?.ui_preferences?.reference_layout) {
      setLayout(user.ui_preferences.reference_layout);
    }
  }, [user]);

  const handleLayoutChange = async (newLayout: GridLayout) => {
    setLayout(newLayout);
    const newPrefs = { reference_layout: newLayout };
    updateUserPreferences(newPrefs);
    if (user?.id) {
      try {
        await api.put(`/employees/${user.id}/preferences`, { ui_preferences: newPrefs });
      } catch (e) {
        console.error("Failed to save layout", e);
      }
    }
  };

  const fetchTables = async () => {
    try {
      setLoading(true);
      const data = await infoTableService.getVisibleTables();
      setTables(data);
    } catch (error) {
      console.error('Failed to fetch info tables:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTables();
  }, []);

  const filteredTables = tables.filter(t => 
    t.name.toLowerCase().includes(searchTerm.toLowerCase()) || 
    (t.description?.toLowerCase() || '').includes(searchTerm.toLowerCase())
  );

  // can create table?
  // Only admin, manager, or employee with can_create_tables
  const canCreate = ['admin', 'manager'].includes(user?.role || '') || user?.can_create_tables;
  const canManagePermissions = user?.role === 'manager' || user?.role === 'admin';

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-2xl font-bold text-foreground flex items-center gap-2">
            <TableIcon className="w-6 h-6 text-primary" />
            References
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            Dynamic data tables for assets, contacts, and custom records
          </p>
        </div>
        
        <div className="flex gap-2">
          {canManagePermissions && (
            <button
              onClick={() => setIsPermissionsModalOpen(true)}
              className="flex items-center gap-2 bg-card text-foreground border border-border px-4 py-2 rounded-lg hover:bg-accent transition-colors shadow-sm"
            >
              <Shield className="w-4 h-4" />
              <span className="hidden sm:inline">Manage Permissions</span>
            </button>
          )}
          {canCreate && (
            <button
              onClick={() => setIsCreateModalOpen(true)}
              className="flex items-center gap-2 bg-primary text-primary-foreground px-4 py-2 rounded-lg hover:bg-primary/90 transition-colors shadow-sm"
            >
              <Plus className="w-4 h-4" />
              Create Table
            </button>
          )}
        </div>
      </div>

      <div className="bg-card rounded-xl shadow-sm border border-border p-4">
        <div className="relative max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search tables..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-border rounded-lg bg-background text-foreground focus:ring-2 focus:ring-primary focus:border-primary transition-all"
          />
        </div>
      </div>

      {loading ? (
        <div className="flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>
      ) : filteredTables.length === 0 ? (
        <div className="text-center py-12 bg-card rounded-xl border border-dashed border-border">
          <TableIcon className="w-12 h-12 text-muted-foreground mx-auto mb-3" />
          <h3 className="text-lg font-medium text-foreground">No tables found</h3>
          <p className="text-sm text-muted-foreground mt-1">
            {searchTerm ? "Try adjusting your search" : "Get started by creating a new table"}
          </p>
        </div>
      ) : (
        <DraggableGrid
          items={filteredTables}
          layout={layout}
          onLayoutChange={handleLayoutChange}
          isSearchActive={searchTerm.length > 0}
          onItemClick={(table) => navigate(`/info-tables/${table.id}`)}
          renderItem={(table) => (
            <div 
              className="bg-card rounded-xl shadow-sm border border-border p-5 hover:shadow-md hover:border-primary/50 transition-all cursor-pointer group flex flex-col h-full"
            >
              <div className="flex justify-between items-start mb-4">
                <div className="bg-primary/10 p-3 rounded-lg text-primary">
                  <FileText className="w-6 h-6" />
                </div>
                <button 
                  onClick={(e) => {
                    e.stopPropagation();
                  }}
                  className="p-1.5 text-muted-foreground hover:text-foreground rounded-md hover:bg-accent opacity-0 group-hover:opacity-100 transition-opacity"
                >
                  <MoreVertical className="w-4 h-4" />
                </button>
              </div>
              
              <h3 className="text-lg font-semibold text-foreground mb-1">
                {table.name}
              </h3>
              
              <p className="text-sm text-muted-foreground line-clamp-2 mb-4 flex-grow">
                {table.description || "No description provided."}
              </p>
              
              <div className="flex items-center justify-between mt-auto pt-4 border-t border-border">
                <div className="flex items-center text-xs text-muted-foreground">
                  <span className="font-medium bg-muted px-2 py-1 rounded-md text-foreground">
                    {table.columns.length} columns
                  </span>
                </div>
                
                <div className="flex items-center text-xs text-muted-foreground gap-1">
                  {table.department_id ? (
                    <><Lock className="w-3 h-3" /> Dept Restricted</>
                  ) : (
                    <><Globe className="w-3 h-3" /> Global</>
                  )}
                </div>
              </div>
            </div>
          )}
        />
      )}

      <CreateTableModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onCreated={(newTable) => {
          setTables([newTable, ...tables]);
          setIsCreateModalOpen(false);
        }}
      />
      
      <TablePermissionsModal
        isOpen={isPermissionsModalOpen}
        onClose={() => setIsPermissionsModalOpen(false)}
      />
    </div>
  );
};

export default InfoTableHub;
