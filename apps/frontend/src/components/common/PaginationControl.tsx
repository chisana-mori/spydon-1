import {
    Pagination,
    PaginationContent,
    PaginationEllipsis,
    PaginationItem,
    PaginationLink,
    PaginationNext,
    PaginationPrevious,
} from "@/components/ui/pagination"

interface PaginationControlProps {
    total: number;
    page: number;
    size: number;
    onPageChange: (page: number) => void;
    onSizeChange?: (size: number) => void;
}

export function PaginationControl({ total, page, size, onPageChange }: PaginationControlProps) {
    const totalPages = Math.ceil(total / size);
    if (totalPages <= 1) return null;

    const renderPageLinks = () => {
        const links = [];
        const maxVisible = 5;

        let start = Math.max(1, page - 2);
        let end = Math.min(totalPages, page + 2);

        if (start === 1) {
            end = Math.min(totalPages, maxVisible);
        }
        if (end === totalPages) {
            start = Math.max(1, totalPages - maxVisible + 1);
        }

        // Always show first page
        if (start > 1) {
            links.push(
                <PaginationItem key={1}>
                    <PaginationLink onClick={() => onPageChange(1)} isActive={page === 1}>
                        1
                    </PaginationLink>
                </PaginationItem>
            );
            if (start > 2) {
                links.push(
                    <PaginationItem key="start-ellipsis">
                        <PaginationEllipsis />
                    </PaginationItem>
                );
            }
        }

        for (let i = start; i <= end; i++) {
            links.push(
                <PaginationItem key={i}>
                    <PaginationLink onClick={() => onPageChange(i)} isActive={page === i}>
                        {i}
                    </PaginationLink>
                </PaginationItem>
            );
        }

        // Always show last page
        if (end < totalPages) {
            if (end < totalPages - 1) {
                links.push(
                    <PaginationItem key="end-ellipsis">
                        <PaginationEllipsis />
                    </PaginationItem>
                );
            }
            links.push(
                <PaginationItem key={totalPages}>
                    <PaginationLink onClick={() => onPageChange(totalPages)} isActive={page === totalPages}>
                        {totalPages}
                    </PaginationLink>
                </PaginationItem>
            );
        }

        return links;
    };

    return (
        <Pagination>
            <PaginationContent>
                <PaginationItem>
                    <PaginationPrevious
                        onClick={() => onPageChange(Math.max(1, page - 1))}
                        // disabled={page === 1} // shadcn PaginationPrevious doesn't have disabled prop easily without custom CSS, but we can make it no-op
                        className={page === 1 ? "pointer-events-none opacity-50" : "cursor-pointer"}
                    />
                </PaginationItem>

                {renderPageLinks()}

                <PaginationItem>
                    <PaginationNext
                        onClick={() => onPageChange(Math.min(totalPages, page + 1))}
                        className={page === totalPages ? "pointer-events-none opacity-50" : "cursor-pointer"}
                    />
                </PaginationItem>
            </PaginationContent>
        </Pagination>
    )
}
