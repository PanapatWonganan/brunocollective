import Link from "next/link";
import styles from "./Footer.module.css";

export default function Footer() {
  return (
    <footer id="contact" className={styles.footer}>
      <div className="wrap">
        <div className={styles.grid}>
          <div className={styles.brandBlock}>
            <div className={styles.mark}>BC</div>
            <div className={styles.name}>Bruno Collective</div>
            <p className={styles.pitch}>
              A heritage house, quietly making clothes for the long quiet of a
              life lived deliberately. Since 1908.
            </p>
          </div>

          <div>
            <h5 className={styles.h5}>Maison</h5>
            <ul className={styles.ul}>
              <li><Link href="/#philosophy">Heritage</Link></li>
              <li><Link href="/#atelier">The Atelier</Link></li>
              <li><Link href="/#journal">Sustainability</Link></li>
              <li><Link href="/#journal">Press</Link></li>
              <li><Link href="/#contact">Careers</Link></li>
            </ul>
          </div>

          <div>
            <h5 className={styles.h5}>Service</h5>
            <ul className={styles.ul}>
              <li><Link href="/shop">Made-to-Measure</Link></li>
              <li><Link href="/#contact">Private Appointment</Link></li>
              <li><Link href="/#contact">Care &amp; Repair</Link></li>
              <li><Link href="/#contact">Shipping</Link></li>
              <li><Link href="/#contact">Contact</Link></li>
            </ul>
          </div>

          <div className={styles.boutiques} id="boutiques">
            <h5 className={styles.h5}>Boutiques</h5>
            <ul className={styles.ul}>
              <li><span className={styles.city}>Milano</span><span className={styles.addr}>Via Gesù 6 · +39 02 760 0900</span></li>
              <li><span className={styles.city}>Paris</span><span className={styles.addr}>14 Rue de Marignan · +33 1 53 75 02</span></li>
              <li><span className={styles.city}>London</span><span className={styles.addr}>38 Mount Street, W1K</span></li>
              <li><span className={styles.city}>New York</span><span className={styles.addr}>695 Madison Avenue, 10065</span></li>
            </ul>
          </div>

          <div>
            <h5 className={styles.h5}>Correspondence</h5>
            <ul className={styles.ul}>
              <li><a href="mailto:atelier@bruno.it">atelier@bruno.it</a></li>
              <li><a href="tel:+390276000900">+39 02 760 0900</a></li>
              <li style={{ marginTop: 18 }}><Link href="/#contact">Press Enquiries</Link></li>
              <li><Link href="/#contact">Private Sales</Link></li>
            </ul>
          </div>
        </div>

        <div className={styles.bottom}>
          <div>© MMXXVI Bruno Collective S.p.A. — All rights reserved.</div>
          <div className={styles.social}>
            <a href="#">Instagram</a>
            <a href="#">Journal</a>
            <a href="#">Pinterest</a>
          </div>
          <div>Bellagio · Solomeo · Milano</div>
        </div>
      </div>
    </footer>
  );
}
