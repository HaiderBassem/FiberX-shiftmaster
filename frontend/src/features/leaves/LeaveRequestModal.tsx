import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { X, Send } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { useTranslation } from 'react-i18next';

interface LeaveRequestModalProps {
  isOpen: boolean;
  onClose: () => void;
  leaveTypes: any[];
  isLoadingTypes: boolean;
  leaveTypeId: string;
  setLeaveTypeId: (id: string) => void;
  startDate: string;
  setStartDate: (date: string) => void;
  endDate: string;
  setEndDate: (date: string) => void;
  startTime: string;
  setStartTime: (time: string) => void;
  endTime: string;
  setEndTime: (time: string) => void;
  reason: string;
  setReason: (reason: string) => void;
  onSubmit: () => void;
  isPending: boolean;
  canSubmit: boolean;
  isHourly: boolean;
  error: string | null;
  remainingText?: string;
}

export const LeaveRequestModal: React.FC<LeaveRequestModalProps> = ({
  isOpen, onClose, leaveTypes, isLoadingTypes, leaveTypeId, setLeaveTypeId,
  startDate, setStartDate, endDate, setEndDate, startTime, setStartTime,
  endTime, setEndTime, reason, setReason, onSubmit, isPending, canSubmit,
  isHourly, error, remainingText
}) => {
  const { t } = useTranslation();
  const selectClass = "w-full h-11 px-4 py-2 rounded-xl bg-muted/50 border-transparent focus:bg-background focus:border-primary text-foreground focus:outline-none focus:ring-2 focus:ring-primary/20 transition-all";
  const inputClass = "w-full h-11 px-4 py-2 rounded-xl bg-muted/50 border-transparent focus:bg-background focus:border-primary text-foreground focus:outline-none focus:ring-2 focus:ring-primary/20 transition-all";

  return (
    <AnimatePresence>
      {isOpen && (
        <>
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
            className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50"
          />
          <motion.div
            initial={{ opacity: 0, scale: 0.95, y: 20 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95, y: 20 }}
            transition={{ type: "spring", duration: 0.5, bounce: 0.3 }}
            className="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-[90%] max-w-md bg-background rounded-3xl shadow-2xl z-50 overflow-hidden border border-white/10"
          >
            <div className="p-6 space-y-6">
              <div className="flex items-center justify-between">
                <h2 className="text-xl font-bold">{t('leaves.new_leave_request')}</h2>
                <button onClick={onClose} className="p-2 rounded-full hover:bg-muted transition-colors">
                  <X className="w-5 h-5 text-muted-foreground" />
                </button>
              </div>

              {error && (
                <div className="p-4 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-sm font-medium">
                  {error}
                </div>
              )}

              <div className="space-y-4">
                <div className="space-y-2">
                  <Label className="text-muted-foreground ml-1">{t('leaves.leave_type')}</Label>
                  <select 
                    className={selectClass} 
                    value={leaveTypeId || (leaveTypes?.[0]?.id || '')} 
                    onChange={(e) => setLeaveTypeId(e.target.value)} 
                    disabled={isLoadingTypes}
                  >
                    {leaveTypes?.map((type: any) => (
                      <option key={type.id} value={type.id}>{type.name_en}</option>
                    ))}
                  </select>
                  {remainingText && (
                    <p className="text-sm font-medium text-emerald-600 mt-2 px-1">
                      {remainingText}
                    </p>
                  )}
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label className="text-muted-foreground ml-1">{isHourly ? t('leaves.date') : t('leaves.start_date')}</Label>
                    <input 
                      type="date" 
                      className={inputClass}
                      value={startDate} 
                      onChange={(e) => {
                        setStartDate(e.target.value);
                        if (isHourly) setEndDate(e.target.value);
                      }} 
                    />
                  </div>
                  {!isHourly && (
                    <div className="space-y-2">
                      <Label className="text-muted-foreground ml-1">{t('leaves.end_date')}</Label>
                      <input 
                        type="date" 
                        className={inputClass}
                        value={endDate} 
                        onChange={(e) => setEndDate(e.target.value)} 
                      />
                    </div>
                  )}
                </div>

                {isHourly && (
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label className="text-muted-foreground ml-1">{t('leaves.from_time')}</Label>
                      <input 
                        type="time" 
                        className={inputClass}
                        value={startTime} 
                        onChange={(e) => setStartTime(e.target.value)} 
                      />
                    </div>
                    <div className="space-y-2">
                      <Label className="text-muted-foreground ml-1">{t('leaves.to_time')}</Label>
                      <input 
                        type="time" 
                        className={inputClass}
                        value={endTime} 
                        onChange={(e) => setEndTime(e.target.value)} 
                      />
                    </div>
                  </div>
                )}

                <div className="space-y-2">
                  <Label className="text-muted-foreground ml-1">{t('leaves.reason')}</Label>
                  <textarea 
                    className="w-full h-28 px-4 py-3 rounded-xl bg-muted/50 border-transparent focus:bg-background focus:border-primary text-foreground focus:outline-none focus:ring-2 focus:ring-primary/20 resize-none transition-all"
                    placeholder={t('leaves.reason_placeholder')}
                    value={reason}
                    onChange={(e) => setReason(e.target.value)}
                  />
                </div>
              </div>

              <Button 
                className="w-full h-12 rounded-xl text-base font-semibold shadow-lg shadow-primary/20 gap-2" 
                onClick={onSubmit} 
                disabled={isPending || !canSubmit}
              >
                <Send className="w-5 h-5" /> {t('leaves.submit_request')}
              </Button>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
};
