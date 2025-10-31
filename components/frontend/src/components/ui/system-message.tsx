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
  // Expect a simple string in data.message; fallback to raw data or JSON.stringify
  let text: string;
  
  if (typeof data?.message === 'string' && data.message) {
    text = data.message;
  } else if (data?.raw) {
    // If we have raw data, try to show it in a more readable way
    text = JSON.stringify(data.raw, null, 2);
  } else if (typeof data === 'string') {
    text = data;
  } else {
    text = JSON.stringify(data ?? {}, null, 2);
  }

  // Compact style: Just small grey text, no card, no avatar
  return (
    <div className={cn("my-1 px-2", className)}>
      <p className="text-xs text-gray-400 italic whitespace-pre-wrap">
        {text}
      </p>
    </div>
  );
};

export default SystemMessage;


