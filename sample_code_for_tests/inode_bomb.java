import java.nio.file.Files;
import java.nio.file.Path;

public class Main {
    public static void main(String[] args) throws Exception {
        for (int i = 0; i < 10000; i++) {
            Files.writeString(Path.of("file_" + i + ".txt"), "");
        }
        System.err.println("inode bomb completed");
    }
}
