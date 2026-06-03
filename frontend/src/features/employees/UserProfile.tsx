import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/features/auth/store';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { CalendarDays, CheckSquare, Clock, User } from 'lucide-react';

export const UserProfile = () => {
  const { user } = useAuthStore();

  const { data: stats, isLoading } = useQuery({
    queryKey: ['profile-stats'],
    queryFn: async () => {
      const response = await api.get('/employees/me/profile-stats');
      return response.data?.data;
    },
  });

  if (isLoading) {
    return <div className="p-8 text-center text-muted-foreground animate-pulse">Loading profile...</div>;
  }

  return (
    <div className="space-y-6 animate-in fade-in slide-in-from-bottom-4 duration-500">
      <div className="flex items-center gap-4">
        <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center border-2 border-primary/20">
          <User className="w-8 h-8 text-primary" />
        </div>
        <div>
          <h2 className="text-2xl font-bold tracking-tight text-foreground">{user?.first_name} {user?.last_name}</h2>
          <p className="text-muted-foreground">{user?.role === 'manager' ? 'Manager' : user?.role === 'team_leader' ? 'Team Leader' : 'Employee'}</p>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card className="hover:shadow-md transition-shadow">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Tasks Completed</CardTitle>
            <CheckSquare className="h-4 w-4 text-emerald-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.completed_tasks || 0}</div>
            <p className="text-xs text-muted-foreground">
              {stats?.active_tasks || 0} tasks currently active
            </p>
          </CardContent>
        </Card>

        <Card className="hover:shadow-md transition-shadow">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Leaves Taken</CardTitle>
            <CalendarDays className="h-4 w-4 text-amber-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.total_leaves_taken || 0}</div>
            <p className="text-xs text-muted-foreground">
              Approved requests this year
            </p>
          </CardContent>
        </Card>

        <Card className="hover:shadow-md transition-shadow">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Hours Logged</CardTitle>
            <Clock className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.worked_hours || 0}</div>
            <p className="text-xs text-muted-foreground">
              Estimated working hours
            </p>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Leave Balances ({new Date().getFullYear()})</CardTitle>
            <CardDescription>Your available quotas for different types of leave.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            {stats?.leave_balances && stats.leave_balances.length > 0 ? (
              stats.leave_balances.map((balance: any) => {
                const percentage = Math.min((balance.used_days / balance.allocated_days) * 100, 100);
                const remaining = balance.allocated_days - balance.used_days;
                
                return (
                  <div key={balance.id} className="space-y-2">
                    <div className="flex justify-between text-sm">
                      <div className="flex items-center gap-2">
                        <span className="w-3 h-3 rounded-full" style={{ backgroundColor: balance.color_code || '#ccc' }} />
                        <span className="font-medium">{balance.leave_type_name_en}</span>
                      </div>
                      <span className="text-muted-foreground">
                        {balance.used_days} / {balance.allocated_days} days used
                      </span>
                    </div>
                    <div className="w-full bg-secondary h-2 rounded-full overflow-hidden">
                      <div 
                        className="h-full rounded-full transition-all duration-500" 
                        style={{ width: `${percentage}%`, backgroundColor: balance.color_code || '#3b82f6' }} 
                      />
                    </div>
                    <p className="text-xs text-right text-muted-foreground font-medium">
                      {remaining} days remaining
                    </p>
                  </div>
                );
              })
            ) : (
              <div className="text-center py-6 text-muted-foreground">
                No leave balances configured for this year.
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
};
