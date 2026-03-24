public class Main {
    public static void main(String[] args) throws Exception {
        String line = "IO_FLOOD:" + "A".repeat(4086);
        while (true) {
            System.err.println(line);
            System.err.flush();
            Thread.sleep(1);
        }
    }
}
