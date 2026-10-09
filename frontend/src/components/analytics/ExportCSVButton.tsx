import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Loader2, Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useQueryClient } from "@tanstack/react-query";
import { downloadFile, downloadErrorKey } from "@/api/download";

interface ExportCSVButtonProps {
  examId: string;
}

export function ExportCSVButton({ examId }: ExportCSVButtonProps) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const qc = useQueryClient();

  async function handleExport() {
    setLoading(true);
    setError(null);
    try {
      await downloadFile(
        qc,
        `/api/v1/admin/exams/${examId}/results/export`,
        `exam-${examId}-results.csv`,
      );
    } catch (err) {
      setError(downloadErrorKey(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-2">
      <Button variant="outline" onClick={handleExport} disabled={loading}>
        {loading ? (
          <>
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            {t("common.loading")}
          </>
        ) : (
          <>
            <Download className="mr-2 h-4 w-4" />
            {t("exam_analytics.export_csv")}
          </>
        )}
      </Button>
      {error && (
        <div
          role="alert"
          className="px-4 py-3 rounded-md text-sm bg-red-50 border border-red-200 text-red-800"
        >
          {t(error)}
        </div>
      )}
    </div>
  );
}
