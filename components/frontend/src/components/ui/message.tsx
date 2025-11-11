"use client";

import React from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { Components } from "react-markdown";

export type MessageRole = "bot" | "user";

export type MessageProps = {
  role: MessageRole;
  content: string;
  isLoading?: boolean;
  avatar?: string;
  name?: string;
  className?: string;
  components?: Components;
  borderless?: boolean;
  actions?: React.ReactNode;
};

const defaultComponents: Components = {
  code: ({
    inline,
    className,
    children,
    ...props
  }: {
    inline?: boolean;
    className?: string;
    children?: React.ReactNode;
  } & React.HTMLAttributes<HTMLElement>) => {
    return inline ? (
      <code
        className="bg-gray-100 px-1 py-0.5 rounded text-xs"
        {...(props as React.HTMLAttributes<HTMLElement>)}
      >
        {children}
      </code>
    ) : (
      <pre className="bg-gray-800 text-gray-100 p-2 rounded text-xs overflow-x-auto">
        <code
          className={className}
          {...(props as React.HTMLAttributes<HTMLElement>)}
        >
          {children}
        </code>
      </pre>
    );
  },
  p: ({ children }) => (
    <p className="text-gray-600 leading-relaxed mb-2 text-sm">{children}</p>
  ),
  h1: ({ children }) => (
    <h1 className="text-lg font-bold text-gray-800 mb-2">{children}</h1>
  ),
  h2: ({ children }) => (
    <h2 className="text-md font-semibold text-gray-800 mb-2">{children}</h2>
  ),
  h3: ({ children }) => (
    <h3 className="text-sm font-medium text-gray-800 mb-1">{children}</h3>
  ),
};

const LOADING_MESSAGES = [
  "Pretending to be productive",
  "Downloading more RAM",
  "Consulting the magic 8-ball",
  "Teaching bugs to behave",
  "Brewing digital coffee",
  "Rolling for initiative",
  "Surfing the data waves",
  "Juggling bits and bytes",
  "Tipping my fedora",
  "Reticulating splines",
];

export const LoadingDots = () => {
  const [messageIndex, setMessageIndex] = React.useState(() =>
    Math.floor(Math.random() * LOADING_MESSAGES.length)
  );

  React.useEffect(() => {
    const intervalId = setInterval(() => {
      setMessageIndex((prevIndex) => (prevIndex + 1) % LOADING_MESSAGES.length);
    }, 8000);
    return () => clearInterval(intervalId);
  }, []);

  return (
    <div className="flex items-center mt-2">
      <svg
        width="24"
        height="8"
        viewBox="0 0 24 8"
        xmlns="http://www.w3.org/2000/svg"
        className="mr-2"
      >
        <style>
          {`
            @keyframes loadingDotPulse {
              0%, 60%, 100% {
                opacity: 0.3;
              }
              30% {
                opacity: 1;
              }
            }
            .loading-dot {
              animation: loadingDotPulse 1.4s infinite ease-in-out;
            }
            .loading-dot-1 {
              animation-delay: 0s;
            }
            .loading-dot-2 {
              animation-delay: 0.2s;
            }
            .loading-dot-3 {
              animation-delay: 0.4s;
            }
          `}
        </style>
        <circle
          className="loading-dot loading-dot-1"
          cx="4"
          cy="4"
          r="3"
          fill="#3b82f6"
        />
        <circle
          className="loading-dot loading-dot-2"
          cx="12"
          cy="4"
          r="3"
          fill="#3b82f6"
        />
        <circle
          className="loading-dot loading-dot-3"
          cx="20"
          cy="4"
          r="3"
          fill="#3b82f6"
        />
      </svg>
      <span className="ml-2 text-xs text-gray-400">{LOADING_MESSAGES[messageIndex]}</span>
    </div>
  );
};

export const Message = React.forwardRef<HTMLDivElement, MessageProps>(
  (
    { role, content, isLoading, className, components, borderless, actions, ...props },
    ref
  ) => {
    const isBot = role === "bot";
    const avatarBg = isBot ? "bg-blue-600" : "bg-green-600";
    const avatarText = isBot ? "AI" : "U";
    const displayName = isBot ? "Claude AI" : "User";

    const avatar = (
      <div className="flex-shrink-0">
      <div
        className={cn(
          "w-8 h-8 rounded-full flex items-center justify-center",
          avatarBg,
          isLoading && "animate-pulse"
        )}
      >
        <span className="text-white text-xs font-semibold">
          {avatarText}
        </span>
      </div>
    </div>
    )

    return (
      <div ref={ref} className={cn("mb-4", className)} {...props}>
        <div className="flex items-start space-x-3">
          {/* Avatar */}
         {isBot ? avatar : null}

          {/* Message Content */}
          <div className="flex-1 min-w-0">
            <div className={cn(borderless ? "p-0" : "bg-white rounded-lg border shadow-sm p-3")}> 
              {/* Header */}
              <div className={cn("flex items-center", borderless ? "mb-1" : "mb-2")}> 
                <Badge
                  variant="outline"
                  className={cn("text-xs", isLoading && "animate-pulse")}
                >
                  {displayName}
                </Badge>
              </div>

              {/* Content */}
              <div className="text-sm text-gray-800">
                {isLoading ? (
                  <div>
                    <div className="text-sm text-gray-600 mb-2">{content}</div>
                    <LoadingDots />
                  </div>
                ) : (
                  <ReactMarkdown
                    remarkPlugins={[remarkGfm]}
                    components={components || defaultComponents}
                  >
                    {content}
                  </ReactMarkdown>
                )}
              </div>

              {actions ? (
                <div className={cn(borderless ? "mt-1" : "mt-3 pt-2 border-t")}>{actions}</div>
              ) : null}
            </div>
          </div>

          {isBot ? null : avatar}
        </div>
      </div>
    );
  }
);

Message.displayName = "Message";
