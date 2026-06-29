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
              A small Thai house, quietly making considered clothing from a love
              of fine cloth and understated luxury. Designed &amp; made in Khon Kaen.
            </p>
          </div>

          <div>
            <h5 className={styles.h5}>The House</h5>
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
            <h5 className={styles.h5}>Atelier</h5>
            <ul className={styles.ul}>
              <li><span className={styles.city}>Khon Kaen</span><span className={styles.addr}>ขอนแก่น · By appointment</span></li>
              <li><span className={styles.city}>Online</span><span className={styles.addr}>Shipping across Thailand · จัดส่งทั่วไทย</span></li>
            </ul>
          </div>

          <div>
            <h5 className={styles.h5}>Correspondence</h5>
            <ul className={styles.ul}>
              <li><a href="mailto:hello@brunocollective.co">hello@brunocollective.co</a></li>
              <li><Link href="/#contact">Private Appointment · นัดหมาย</Link></li>
              <li style={{ marginTop: 18 }}><Link href="/#contact">Press Enquiries</Link></li>
              <li><Link href="/#contact">Wholesale</Link></li>
            </ul>
          </div>
        </div>

        <div className={styles.bottom}>
          <div>© 2026 Bruno Collective — All rights reserved.</div>
          <div className={styles.social}>
            <a href="#">Instagram</a>
            <a href="#">Journal</a>
            <a href="#">LINE</a>
          </div>
          <div>Designed &amp; made in Khon Kaen, Thailand</div>
        </div>
      </div>
    </footer>
  );
}
