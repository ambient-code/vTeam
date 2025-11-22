import React from "react";
import { cn } from "@/lib/utils";

type SystemMessageData = {
  message?: string;
  [key: string]: unknown;
};

export type SystemMessageProps = {
  subtype?: string;
  data: SystemMessageData;
  className?: string;
  borderless?: boolean;
};

export const SystemMessage: React.FC<SystemMessageProps> = ({ data, className }) => {
  // Expect a simple string in data.message; fallback to JSON.stringify
  const text: string = typeof (data?.message) === 'string' ? data.message : (typeof data === 'string' ? data : JSON.stringify(data ?? {}, null, 2));

  // Compact style: Just small grey text, no card, no avatar
  return (
    <div className={cn("my-1 px-2", className)}>
      <p className="text-xs text-muted-foreground/60 italic">
        {text}
      </p>
    </div>
  );
};

export default SystemMessage;


