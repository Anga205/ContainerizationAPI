import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;

public class Main {
    public static void main(String[] args) throws Exception {
        Path p = Path.of("/root/test.txt");
        try {
            System.out.print(Files.readString(p, StandardCharsets.UTF_8));
        } catch (Exception ex) {
            System.err.println("No such file or directory: " + ex.getMessage());
            System.exit(1);
        }
    }
}
