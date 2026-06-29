import React, { useState } from 'react';
import { useAuthStore } from '@/store/authStore';
import { useMyLinks } from '@/hooks/useModuleAccess';
import { Search, Link as LinkIcon, ExternalLink } from 'lucide-react';
import { getLinkIcon, getLinkTheme } from '@/components/icons';
import { openExternalLink } from '@/lib/utils';

export function ExternalToolsList() {
  const { user } = useAuthStore();
  const { data: myModules, isLoading } = useMyLinks();
  const [searchTerm, setSearchTerm] = useState('');

  if (isLoading) {
    return <div className="p-8 text-center text-muted-foreground">Loading...</div>;
  }

  const links = myModules || [];
  const filteredLinks = links.filter((link) =>
    link.title.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-2xl font-bold text-foreground">External Tools</h1>
          <p className="text-muted-foreground text-sm mt-1">
            Access your external tools and modules
          </p>
        </div>
      </div>

      <div className="bg-card rounded-xl shadow-sm border border-border p-4">
        <div className="relative max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search tools..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-border rounded-lg bg-background text-foreground focus:ring-2 focus:ring-primary focus:border-primary transition-all"
          />
        </div>
      </div>

      {filteredLinks.length === 0 ? (
        <div className="bg-card rounded-xl border border-border p-12 text-center">
          <div className="w-16 h-16 bg-muted rounded-full flex items-center justify-center mx-auto mb-4">
            <LinkIcon className="w-8 h-8 text-muted-foreground" />
          </div>
          <h3 className="text-lg font-medium text-foreground mb-1">No Tools Found</h3>
          <p className="text-muted-foreground max-w-sm mx-auto">
            {searchTerm
              ? "No external tools match your search criteria."
              : "No external tools have been assigned to your department yet."}
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {filteredLinks.map((link) => {
            const theme = getLinkTheme(link.icon_name);
            const Icon = getLinkIcon(link.icon_name);

            return (
              <div
                key={link.id}
                onClick={() => openExternalLink(link, user?.role)}
                className="bg-card rounded-xl border border-border p-6 hover:shadow-md hover:border-primary/50 transition-all cursor-pointer group flex flex-col h-full"
              >
                <div className="flex items-start justify-between mb-4">
                  <div
                    className="w-12 h-12 rounded-lg flex items-center justify-center group-hover:scale-105 transition-transform"
                    style={{ background: `linear-gradient(135deg, ${theme.c1} 0%, ${theme.c2} 100%)` }}
                  >
                    <Icon className="w-6 h-6 text-white" />
                  </div>
                  <div className="p-2 bg-muted/50 rounded-full group-hover:bg-primary/10 transition-colors">
                    <ExternalLink className="w-4 h-4 text-muted-foreground group-hover:text-primary" />
                  </div>
                </div>

                <h3
                  className="text-lg font-bold text-foreground mb-2 line-clamp-2"
                  style={{ color: theme.c1 }}
                >
                  {link.title}
                </h3>
                {link.description && (
                  <p className="text-sm text-muted-foreground line-clamp-3">
                    {link.description}
                  </p>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
