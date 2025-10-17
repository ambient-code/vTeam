import { Loader2 } from 'lucide-react';

export default function RootLoading() {
  return (
    <div className="flex items-center justify-center min-h-screen">
      <div className="flex flex-col items-center gap-4">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        {/* <p className="text-sm text-muted-foreground">Loading...</p> */}
      </div>
    </div>
  );
}
