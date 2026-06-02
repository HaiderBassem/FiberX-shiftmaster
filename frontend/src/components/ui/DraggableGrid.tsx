import React, { useState, useMemo, useEffect } from 'react';
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragOverlay,
  useDroppable,
} from '@dnd-kit/core';
import type { DragEndEvent, DragStartEvent } from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  rectSortingStrategy,
  useSortable
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { Folder as FolderIcon, X, Edit2, ChevronDown, ChevronUp, GripHorizontal } from 'lucide-react';

export interface LayoutFolder {
  id: string;
  name: string;
  itemIds: string[];
}

export interface GridLayout {
  folders: Record<string, LayoutFolder>;
  order: string[]; // item IDs and folder IDs in order
}

interface DraggableGridProps {
  items: any[]; // The raw items (tables or docs)
  layout: GridLayout;
  onLayoutChange: (layout: GridLayout) => void;
  renderItem: (item: any) => React.ReactNode;
  onItemClick: (item: any) => void;
  isSearchActive: boolean;
}

const SortableItem = ({ id, children }: { id: string, children: React.ReactNode }) => {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
    zIndex: isDragging ? 999 : 'auto',
  };

  return (
    <div ref={setNodeRef} style={style} {...attributes} {...listeners} className="relative group cursor-grab active:cursor-grabbing h-full">
      {children}
    </div>
  );
};

const EmptyFolderDropZone = ({ id }: { id: string }) => {
  const { isOver, setNodeRef } = useDroppable({ id });

  return (
    <div
      ref={setNodeRef}
      className={`text-center py-8 text-sm border-2 border-dashed rounded-lg transition-colors ${
        isOver ? 'bg-indigo-50 border-indigo-400 text-indigo-700' : 'bg-gray-50 dark:bg-gray-800/50 text-gray-400 border-gray-200 dark:border-gray-700'
      }`}
    >
      Drop items here to add them to this folder
    </div>
  );
};

export const DraggableGrid: React.FC<DraggableGridProps> = ({
  items,
  layout,
  onLayoutChange,
  renderItem,
  onItemClick,
  isSearchActive,
}) => {
  const [activeId, setActiveId] = useState<string | null>(null);
  const [expandedFolders, setExpandedFolders] = useState<Record<string, boolean>>({});
  const [editingFolderId, setEditingFolderId] = useState<string | null>(null);
  const [folderNameInput, setFolderNameInput] = useState('');

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 5,
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  // Initialize layout if empty or sync with items
  const normalizedLayout = useMemo(() => {
    if (!layout || !layout.order) {
      return { folders: {} as Record<string, LayoutFolder>, order: items.map(i => i.id) };
    }
    
    const existingIds = new Set(items.map(i => i.id));
    const newOrder = [...layout.order];
    const newFolders: Record<string, LayoutFolder> = { ...layout.folders };

    // Remove deleted items from order and folders
    const orderWithoutDeleted = newOrder.filter(id => {
      if (newFolders[id]) return true; // Keep folders
      return existingIds.has(id); // Keep existing items
    });

    // Clean up folders
    for (const fId in newFolders) {
      const folder = newFolders[fId];
      const validItems = folder.itemIds ? folder.itemIds.filter(id => existingIds.has(id)) : [];
      if (validItems.length !== (folder.itemIds?.length || 0)) {
        newFolders[fId] = { ...folder, itemIds: validItems };
      }
    }

    // Add new items not in layout
    const itemsInLayout = new Set<string>();
    orderWithoutDeleted.forEach(id => {
      if (newFolders[id]) {
        newFolders[id].itemIds.forEach(itemId => itemsInLayout.add(itemId));
      } else {
        itemsInLayout.add(id);
      }
    });

    items.forEach(item => {
      if (!itemsInLayout.has(item.id)) {
        orderWithoutDeleted.push(item.id);
      }
    });

    return { folders: newFolders, order: orderWithoutDeleted };
  }, [items, layout]);

  useEffect(() => {
    // If layout is completely empty, initialize it and notify parent
    if (!layout || !layout.order || layout.order.length === 0) {
       if (items.length > 0) {
         onLayoutChange({ folders: {}, order: items.map(i => i.id) });
       }
    }
  }, [items, layout, onLayoutChange]);

  const handleDragStart = (event: DragStartEvent) => {
    setActiveId(event.active.id as string);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    setActiveId(null);
    const { active, over } = event;

    if (!over) return;

    const activeIdStr = active.id as string;
    let overIdStr = over.id as string;

    if (activeIdStr === overIdStr) return;

    const newLayout = { ...normalizedLayout };
    newLayout.folders = { ...newLayout.folders };
    newLayout.order = [...newLayout.order];

    const isItemInFolder = (id: string) => {
      for (const fId in newLayout.folders) {
        if (newLayout.folders[fId].itemIds.includes(id)) return fId;
      }
      return null;
    };

    let targetFolderId: string | null = null;
    let isDroppingIntoEmptyFolder = false;

    if (overIdStr.endsWith('-empty-dropzone')) {
      targetFolderId = overIdStr.replace('-empty-dropzone', '');
      isDroppingIntoEmptyFolder = true;
    }

    const activeParent = isItemInFolder(activeIdStr);
    const overParent = isDroppingIntoEmptyFolder ? targetFolderId : isItemInFolder(overIdStr);

    // If moving within the SAME parent
    if (activeParent === overParent && !isDroppingIntoEmptyFolder) {
      if (activeParent) {
        // Reorder inside folder
        const folder = newLayout.folders[activeParent];
        const oldIndex = folder.itemIds.indexOf(activeIdStr);
        const newIndex = folder.itemIds.indexOf(overIdStr);
        newLayout.folders[activeParent] = {
          ...folder,
          itemIds: arrayMove(folder.itemIds, oldIndex, newIndex)
        };
      } else {
        // Reorder at root
        const oldIndex = newLayout.order.indexOf(activeIdStr);
        const newIndex = newLayout.order.indexOf(overIdStr);
        newLayout.order = arrayMove(newLayout.order, oldIndex, newIndex);
      }
      onLayoutChange(newLayout);
      return;
    }

    // Moving ACROSS parents
    // Remove active from old parent
    if (activeParent) {
      const folder = newLayout.folders[activeParent];
      newLayout.folders[activeParent] = {
        ...folder,
        itemIds: folder.itemIds.filter(id => id !== activeIdStr)
      };
    } else {
      newLayout.order = newLayout.order.filter(id => id !== activeIdStr);
    }

    // Add active to new parent
    if (overParent) {
      // Adding to a folder
      const folder = newLayout.folders[overParent];
      const newItems = [...folder.itemIds];
      if (isDroppingIntoEmptyFolder) {
        newItems.push(activeIdStr);
      } else {
        const overIndex = folder.itemIds.indexOf(overIdStr);
        newItems.splice(overIndex >= 0 ? overIndex : newItems.length, 0, activeIdStr);
      }
      newLayout.folders[overParent] = {
        ...folder,
        itemIds: newItems
      };
    } else {
      // Adding to root
      const newOrder = [...newLayout.order];
      const overIndex = newOrder.indexOf(overIdStr);
      newOrder.splice(overIndex >= 0 ? overIndex : newOrder.length, 0, activeIdStr);
      newLayout.order = newOrder;
    }

    onLayoutChange(newLayout);
  };

  const createFolder = () => {
    const newFolderId = `folder-${Date.now()}`;
    const newLayout = { ...normalizedLayout };
    newLayout.folders = {
      ...newLayout.folders,
      [newFolderId]: { id: newFolderId, name: 'New Folder', itemIds: [] }
    };
    newLayout.order = [newFolderId, ...newLayout.order];
    onLayoutChange(newLayout);
    setEditingFolderId(newFolderId);
    setFolderNameInput('New Folder');
  };

  const renameFolder = (folderId: string) => {
    if (!folderNameInput.trim()) return;
    const newLayout = { ...normalizedLayout };
    newLayout.folders = {
      ...newLayout.folders,
      [folderId]: { ...newLayout.folders[folderId], name: folderNameInput }
    };
    onLayoutChange(newLayout);
    setEditingFolderId(null);
  };

  const deleteFolder = (folderId: string) => {
    const newLayout = { ...normalizedLayout };
    const folder = newLayout.folders[folderId];
    // Move items back to root
    const folderIndex = newLayout.order.indexOf(folderId);
    newLayout.order.splice(folderIndex, 1, ...folder.itemIds);
    delete newLayout.folders[folderId];
    onLayoutChange(newLayout);
  };

  const removeFromFolder = (folderId: string, itemId: string) => {
    const newLayout = { ...normalizedLayout };
    const folder = newLayout.folders[folderId];
    folder.itemIds = folder.itemIds.filter(id => id !== itemId);
    newLayout.order.push(itemId);
    if (folder.itemIds.length === 0) {
      delete newLayout.folders[folderId];
      newLayout.order = newLayout.order.filter(id => id !== folderId);
    }
    onLayoutChange(newLayout);
  };

  const getItemMap = () => {
    const map: Record<string, any> = {};
    items.forEach(i => map[i.id] = i);
    return map;
  };
  const itemMap = getItemMap();

  if (isSearchActive) {
    // Flattened view for search
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {items.map(item => (
          <div key={item.id} onClick={() => onItemClick(item)}>
            {renderItem(item)}
          </div>
        ))}
      </div>
    );
  }

  return (
    <div>
      <div className="flex justify-end mb-4">
        <button 
          onClick={createFolder}
          className="flex items-center gap-1 text-sm bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 px-3 py-1.5 rounded-md hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
        >
          <FolderIcon className="w-4 h-4" />
          Create Folder
        </button>
      </div>

      <DndContext 
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
      >
        <SortableContext items={normalizedLayout.order} strategy={rectSortingStrategy}>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {normalizedLayout.order.map((id) => {
              const folder = normalizedLayout.folders[id];
              if (folder) {
                const isExpanded = expandedFolders[id];
                return (
                  <div key={id} className="col-span-1 md:col-span-2 lg:col-span-3">
                    <SortableItem id={id}>
                      <div className="bg-gray-50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700 rounded-xl overflow-hidden mb-2 shadow-sm">
                        <div 
                          className="flex items-center justify-between p-4 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700/50 transition-colors"
                        >
                          <div className="flex items-center gap-3 flex-1" onClick={() => setExpandedFolders(prev => ({...prev, [id]: !prev[id]}))}>
                            <div className="text-gray-400 hover:text-gray-600 cursor-grab">
                               <GripHorizontal className="w-5 h-5" />
                            </div>
                            <FolderIcon className="w-5 h-5 text-indigo-500" fill="currentColor" fillOpacity={0.2} />
                            
                            {editingFolderId === id ? (
                              <div className="flex items-center gap-2" onClick={e => e.stopPropagation()}>
                                <input 
                                  autoFocus
                                  type="text" 
                                  value={folderNameInput}
                                  onChange={e => setFolderNameInput(e.target.value)}
                                  onKeyDown={e => e.key === 'Enter' && renameFolder(id)}
                                  className="border border-indigo-300 rounded px-2 py-1 text-sm text-gray-900 bg-white"
                                />
                                <button onClick={() => renameFolder(id)} className="text-indigo-600 text-sm font-medium">Save</button>
                              </div>
                            ) : (
                              <h3 className="font-medium text-gray-900 dark:text-white flex-1 flex items-center gap-2">
                                {folder.name}
                                <span className="text-xs font-normal text-gray-500 bg-gray-200 dark:bg-gray-700 px-2 py-0.5 rounded-full">
                                  {folder.itemIds.length}
                                </span>
                              </h3>
                            )}
                          </div>
                          
                          <div className="flex items-center gap-2">
                            <button onClick={(e) => { e.stopPropagation(); setEditingFolderId(id); setFolderNameInput(folder.name); }} className="p-1.5 text-gray-400 hover:text-gray-600 rounded-md">
                              <Edit2 className="w-4 h-4" />
                            </button>
                            <button onClick={(e) => { e.stopPropagation(); deleteFolder(id); }} className="p-1.5 text-gray-400 hover:text-red-500 rounded-md">
                              <X className="w-4 h-4" />
                            </button>
                            <button onClick={() => setExpandedFolders(prev => ({...prev, [id]: !prev[id]}))} className="p-1.5 text-gray-400 hover:text-gray-600 rounded-md">
                              {isExpanded ? <ChevronUp className="w-5 h-5" /> : <ChevronDown className="w-5 h-5" />}
                            </button>
                          </div>
                        </div>
                        
                        {isExpanded && (
                          <div className="p-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50/50 dark:bg-gray-900/30">
                            {folder.itemIds.length === 0 ? (
                              <EmptyFolderDropZone id={`${id}-empty-dropzone`} />
                            ) : (
                              <SortableContext items={folder.itemIds} strategy={rectSortingStrategy}>
                                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                                  {folder.itemIds.map(itemId => {
                                    const item = itemMap[itemId];
                                    if (!item) return null;
                                    return (
                                      <SortableItem key={itemId} id={itemId}>
                                        <div className="relative group">
                                          <div onClick={() => onItemClick(item)}>
                                            {renderItem(item)}
                                          </div>
                                          <button 
                                            onClick={(e) => { e.stopPropagation(); removeFromFolder(id, itemId); }}
                                            className="absolute top-2 right-2 p-1.5 bg-white dark:bg-gray-800 text-gray-400 hover:text-red-500 rounded-md opacity-0 group-hover:opacity-100 shadow-sm border border-gray-200 dark:border-gray-700 transition-opacity z-10"
                                            title="Remove from folder"
                                          >
                                            <X className="w-4 h-4" />
                                          </button>
                                        </div>
                                      </SortableItem>
                                    );
                                  })}
                                </div>
                              </SortableContext>
                            )}
                          </div>
                        )}
                      </div>
                    </SortableItem>
                  </div>
                );
              }

              // Normal root item
              const item = itemMap[id];
              if (!item) return null;
              return (
                <SortableItem key={id} id={id}>
                  <div onClick={() => onItemClick(item)}>
                    {renderItem(item)}
                  </div>
                </SortableItem>
              );
            })}
          </div>
        </SortableContext>
        
        {/* Simple drag overlay */}
        <DragOverlay>
          {activeId ? (
            <div className="opacity-80 scale-105 transition-transform bg-white rounded-xl shadow-xl">
              {/* If it's a folder being dragged */}
              {normalizedLayout.folders[activeId] ? (
                 <div className="p-4 border border-indigo-200 rounded-xl bg-indigo-50 flex items-center gap-3">
                   <FolderIcon className="w-6 h-6 text-indigo-500" />
                   <span className="font-medium">{normalizedLayout.folders[activeId].name}</span>
                 </div>
              ) : (
                 /* If it's an item, we can't render the exact item easily here without duplicating renderItem, so we just show a generic card */
                 itemMap[activeId] ? (
                   <div className="pointer-events-none">
                     {renderItem(itemMap[activeId])}
                   </div>
                 ) : null
              )}
            </div>
          ) : null}
        </DragOverlay>
      </DndContext>
    </div>
  );
};
