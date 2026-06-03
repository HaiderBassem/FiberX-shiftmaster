import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { X, Send } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';

interface SwapRequestModalProps {
  isOpen: boolean;
  onClose: () => void;
  employees: any[];
  targetEmployeeId: string;
  setTargetEmployeeId: (id: string) => void;
  shiftDate: string;
  setShiftDate: (date: string) => void;
  reason: string;
  setReason: (reason: string) => void;
  onSubmit: () => void;
  isPending: boolean;
  canSubmit: boolean;
  error: string | null;
}

export const SwapRequestModal: React.FC<SwapRequestModalProps> = ({
  isOpen, onClose, employees, targetEmployeeId, setTargetEmployeeId,
  shiftDate, setShiftDate, reason, setReason, onSubmit, isPending, canSubmit, error
}) => {
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
                <h2 className="text-xl font-bold">Request Shift Swap</h2>
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
                  <Label className="text-muted-foreground ml-1">Swap With Colleague</Label>
                  <select 
                    className={selectClass} 
                    value={targetEmployeeId} 
                    onChange={(e) => setTargetEmployeeId(e.target.value)}
                  >
                    <option value="">Select colleague...</option>
                    {employees?.map((emp: any) => (
                      <option key={emp.id} value={emp.id}>{emp.first_name} {emp.last_name}</option>
                    ))}
                  </select>
                </div>

                <div className="space-y-2">
                  <Label className="text-muted-foreground ml-1">Your Shift Date</Label>
                  <input 
                    type="date" 
                    className={inputClass}
                    value={shiftDate} 
                    onChange={(e) => setShiftDate(e.target.value)} 
                  />
                </div>

                <div className="space-y-2">
                  <Label className="text-muted-foreground ml-1">Reason</Label>
                  <textarea 
                    className="w-full h-28 px-4 py-3 rounded-xl bg-muted/50 border-transparent focus:bg-background focus:border-primary text-foreground focus:outline-none focus:ring-2 focus:ring-primary/20 resize-none transition-all"
                    placeholder="Provide a clear reason for your swap request..."
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
                <Send className="w-5 h-5" /> Send Swap Request
              </Button>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
};
