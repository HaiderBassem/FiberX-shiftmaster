import { useState } from 'react';
import { createPortal } from 'react-dom';
import { useMutation } from '@tanstack/react-query';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle, CardDescription, CardFooter } from '@/components/ui/card';
import { Key } from 'lucide-react';
import api from '@/lib/api';

interface ChangePasswordModalProps {
  isOpen: boolean;
  onClose: () => void;
  employeeId: string;
  requireOldPassword: boolean;
}

export const ChangePasswordModal = ({ isOpen, onClose, employeeId, requireOldPassword }: ChangePasswordModalProps) => {
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: async () => {
      setError(null);
      if (newPassword !== confirmPassword) {
        throw new Error("New passwords do not match.");
      }
      if (newPassword.length < 8) {
        throw new Error("Password must be at least 8 characters long.");
      }

      await api.put(`/employees/${employeeId}/password`, {
        old_password: requireOldPassword ? oldPassword : "",
        new_password: newPassword,
      });
    },
    onSuccess: () => {
      handleClose();
    },
    onError: (err: any) => {
      setError(err?.response?.data?.error || err?.message || 'Failed to change password');
    },
  });

  const handleClose = () => {
    setOldPassword('');
    setNewPassword('');
    setConfirmPassword('');
    setError(null);
    onClose();
  };

  if (!isOpen) return null;

  return createPortal(
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-4 sm:p-0">
      <div className="fixed inset-0 bg-background/80 backdrop-blur-sm" onClick={handleClose} />
      <Card className="relative w-full max-w-md shadow-lg border-border animate-in fade-in zoom-in-95 duration-200">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Key className="w-5 h-5 text-primary" />
            {requireOldPassword ? 'Change Password' : 'Reset Employee Password'}
          </CardTitle>
          <CardDescription>
            {requireOldPassword 
              ? 'Enter your current password to set a new one.'
              : 'Set a new password for this employee.'}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {error && (
            <div className="p-3 text-sm text-destructive bg-destructive/10 border border-destructive/20 rounded-xl">
              {error}
            </div>
          )}

          {requireOldPassword && (
            <div className="space-y-2">
              <Label htmlFor="old_password">Current Password</Label>
              <Input
                id="old_password"
                type="password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                placeholder="Enter current password"
              />
            </div>
          )}

          <div className="space-y-2">
            <Label htmlFor="new_password">New Password</Label>
            <Input
              id="new_password"
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              placeholder="Minimum 8 characters"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="confirm_password">Confirm New Password</Label>
            <Input
              id="confirm_password"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              placeholder="Re-enter new password"
            />
          </div>
        </CardContent>
        <CardFooter className="flex justify-end gap-2">
          <Button variant="outline" onClick={handleClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button 
            onClick={() => mutation.mutate()} 
            disabled={mutation.isPending || !newPassword || !confirmPassword || (requireOldPassword && !oldPassword)}
          >
            {mutation.isPending ? 'Saving...' : 'Save Password'}
          </Button>
        </CardFooter>
      </Card>
    </div>,
    document.body
  );
};
