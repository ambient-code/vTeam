import { TableSkeleton } from '@/components/skeletons';

export default function KeysLoading() {
  return <TableSkeleton rows={5} columns={3} />;
}
