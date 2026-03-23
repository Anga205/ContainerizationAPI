import java.util.ArrayList;
import java.util.List;

public class Main {
    public static void main(String[] args) throws Exception {
        List<byte[]> chunks = new ArrayList<>();
        while (true) {
            chunks.add(new byte[1024 * 1024]);
        }
    }
}
