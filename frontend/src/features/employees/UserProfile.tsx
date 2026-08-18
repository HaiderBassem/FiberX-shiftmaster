import React, { useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { CalendarDays, CheckSquare, Clock, User } from 'lucide-react';
import { assetUrl } from '@/lib/assets';

export const UserProfile = () => {
  const { t, i18n } = useTranslation();
  const isAr = i18n.language?.startsWith('ar');
  const { user, updateProfileImage } = useAuthStore();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);

  const { data: stats, isLoading } = useQuery({
    queryKey: ['profile-stats'],
    queryFn: async () => {
      const response = await api.get('/employees/me/profile-stats');
      return response.data?.data;
    },
  });

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const formData = new FormData();
    formData.append('profile_picture', file);

    try {
      setUploading(true);
      const res = await api.post('/employees/me/profile-picture', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      if (res.data?.success && res.data?.data?.profile_image) {
        updateProfileImage(res.data.data.profile_image);
      }
    } catch (error) {
      console.error('Failed to upload image', error);
      alert(t('profile.upload_failed'));
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = '';
    }
  };

  if (isLoading) {
    return <div className="p-8 text-center text-muted-foreground animate-pulse">{t('profile.loading')}</div>;
  }

  return (
    <div className="space-y-6 animate-in fade-in slide-in-from-bottom-4 duration-500">
      <div className="flex items-center gap-4">
        <div 
          className="relative w-24 h-24 rounded-full bg-primary/10 flex items-center justify-center border-2 border-primary/20 overflow-hidden group cursor-pointer shrink-0"
          onClick={() => fileInputRef.current?.click()}
        >
          {user?.profile_image ? (
            <img 
              src={assetUrl(user.profile_image)} 
              alt={t('profile.photo_alt')}
              className="w-full h-full object-cover" 
            />
          ) : (
            <User className="w-12 h-12 text-primary/50" />
          )}
          
          <div className="absolute inset-0 bg-black/50 flex flex-col items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
            <span className="text-xs text-white font-medium">{uploading ? t('profile.uploading') : t('profile.change_photo')}</span>
          </div>
        </div>
        <input 
          type="file" 
          ref={fileInputRef} 
          onChange={handleImageUpload} 
          accept="image/*" 
          className="hidden" 
        />
        <div>
          <h2 className="text-2xl font-bold tracking-tight text-foreground">{user?.first_name} {user?.last_name}</h2>
          <p className="text-muted-foreground">{user?.role === 'manager' ? t('profile.role_manager') : user?.role === 'team_leader' ? t('profile.role_team_leader') : t('profile.role_employee')}</p>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card className="hover:shadow-md transition-shadow">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">{t('profile.tasks_completed')}</CardTitle>
            <CheckSquare className="h-4 w-4 text-emerald-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.completed_tasks || 0}</div>
            <p className="text-xs text-muted-foreground">
              {stats?.active_tasks || 0} {t('profile.tasks_active')}
            </p>
          </CardContent>
        </Card>

        <Card className="hover:shadow-md transition-shadow">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">{t('profile.total_leaves')}</CardTitle>
            <CalendarDays className="h-4 w-4 text-amber-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.total_leaves_taken || 0}</div>
            <p className="text-xs text-muted-foreground">
              {t('profile.approved_this_year')}
            </p>
          </CardContent>
        </Card>

        <Card className="hover:shadow-md transition-shadow">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">{t('profile.hourly_leaves')}</CardTitle>
            <Clock className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.total_hourly_leaves_taken || 0}</div>
            <p className="text-xs text-muted-foreground">
              {t('profile.approved_hourly')}
            </p>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{t('profile.leave_balances', { year: new Date().getFullYear() })}</CardTitle>
            <CardDescription>{t('profile.leave_balances_desc')}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            {stats?.leave_balances && stats.leave_balances.length > 0 ? (
              stats.leave_balances
                .filter((b: any) => b.reset_cycle === 'annual' || b.month === new Date().getMonth() + 1)
                .map((balance: any) => {
                  const percentage = Math.min(((balance.used_amount + (balance.pending_amount || 0)) / balance.allocated_amount) * 100, 100) || 0;
                  const remaining = balance.allocated_amount - balance.used_amount - (balance.pending_amount || 0);
                  
                  return (
                    <div key={balance.id} className="space-y-2">
                      <div className="flex justify-between text-sm">
                        <div className="flex items-center gap-2">
                          <span className="w-3 h-3 rounded-full" style={{ backgroundColor: balance.color_code || '#ccc' }} />
                          <span className="font-medium">
                            {(isAr && balance.leave_type_name_ar) || balance.leave_type_name_en} {balance.reset_cycle === 'monthly' && `(${t('profile.this_month')})`}
                          </span>
                        </div>
                        <span className="text-muted-foreground">
                          <span dir="ltr">{balance.used_amount} / {balance.allocated_amount}</span>{' '}
                          {t(balance.unit === 'hours' ? 'profile.hours_used' : 'profile.days_used')}
                          {balance.pending_amount ? (
                            <> (<span dir="ltr">+{balance.pending_amount}</span> {t('profile.pending')})</>
                          ) : null}
                        </span>
                      </div>
                      <div className="w-full bg-secondary h-2 rounded-full overflow-hidden">
                        <div 
                          className="h-full rounded-full transition-all duration-500" 
                          style={{ width: `${percentage}%`, backgroundColor: balance.color_code || '#3b82f6' }} 
                        />
                      </div>
                      <p className="text-xs text-right text-muted-foreground font-medium">
                        <span dir="ltr">{remaining}</span> {t(balance.unit === 'hours' ? 'profile.hours_remaining' : 'profile.days_remaining')}
                      </p>
                    </div>
                  );
                })
            ) : (
              <div className="text-center py-6 text-muted-foreground">
                {t('profile.no_balances')}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
};
